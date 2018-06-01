package server

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gabstv/manners"
	"github.com/gabstv/sandpiper/pathtree"
	"github.com/gabstv/sandpiper/route"
	"github.com/gabstv/sandpiper/util"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// Server is the structure that controls, routes and certificates.
type Server interface {
	Add(r route.Route) error
	Run() error
	Close()
	Init()
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type sServer struct {
	Cfg         Config
	trieDomains *pathtree.Trie
	domains     map[string]*route.Route
	Logger      *log.Logger
	closeChan   chan os.Signal
	htps        *http.Server
}

// Default starts a server with the default configuration options
func Default(cfg *Config) Server {
	s := &sServer{}
	if cfg != nil {
		s.Cfg = *cfg
	}
	s.trieDomains = pathtree.NewTrie(".")
	s.domains = make(map[string]*route.Route, 0)
	s.Logger = log.New(os.Stderr, "[sp server] ", log.LstdFlags)
	return s
}

func (s *sServer) startAPI(ctx context.Context) error {
	if s.Cfg.APIListen == "" {
		return nil
	}
	go runAPIV1(ctx, s, s.Cfg.APIListen, s.Cfg.APIKey, s.Cfg.Debug)
	if s.Cfg.APIDomain != "" {
		s.Add(route.Route{
			Autocert: s.Cfg.APIDomainAutocert,
			Domain:   s.Cfg.APIDomain,
			Server: route.RouteServer{
				OutAddress:  s.Cfg.APIListen,
				OutConnType: route.HTTP,
			},
			WsCFG: util.WsConfig{
				Enabled: false,
			},
		})
	}
	return nil
}

func (s *sServer) Add(r route.Route) error {
	rr := &route.Route{}
	*rr = r
	if rr.WsCFG.ReadBufferSize == 0 {
		rr.WsCFG.ReadBufferSize = 2048
	}
	if rr.WsCFG.WriteBufferSize == 0 {
		rr.WsCFG.WriteBufferSize = 2048
	}
	if rr.WsCFG.ReadDeadlineSeconds == 0 {
		rr.WsCFG.ReadDeadlineSeconds = time.Second * 60
	} else {
		if rr.WsCFG.ReadDeadlineSeconds < time.Millisecond {
			rr.WsCFG.ReadDeadlineSeconds = time.Duration(rr.WsCFG.ReadDeadlineSeconds) * time.Second
		}
	}
	err := s.trieDomains.Add(r.Domain, rr)
	if err != nil {
		return err
	}
	s.domains[r.Domain] = rr
	s.updateCertificates()
	return nil
}

func (s *sServer) updateCertificates() *autocert.Manager {
	autocdomains := make([]string, 0)
	for _, v := range s.domains {
		if v.Autocert {
			autocdomains = append(autocdomains, v.Domain)
		}
	}
	if len(autocdomains) < 1 {
		return nil
	}
	var m *autocert.Manager
	cpath := "/tmp/sandpiper"
	if s.Cfg.CachePath != "" {
		cpath = s.Cfg.CachePath
	}
	dcache := autocert.DirCache(cpath)
	m = &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(autocdomains...),
		Cache:      &dcache,
	}
	if s.Cfg.LetsEncryptURL != "" {
		m.Client = &acme.Client{DirectoryURL: s.Cfg.LetsEncryptURL}
	}

	certs := make(map[string]tls.Certificate)
	for k, v := range s.domains {
		if !v.Autocert && len(v.Certificate.KeyFile) > 0 && len(v.Certificate.CertFile) > 0 {
			ncert, err := tls.LoadX509KeyPair(v.Certificate.CertFile, v.Certificate.KeyFile)
			if err != nil {
				s.Logger.Println("Error loading certificate for", k, err.Error())
			} else {
				certs[k] = ncert
			}
		}
	}

	getcertfn := func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		if dom, ok := certs[clientHello.ServerName]; ok {
			return &dom, nil
		}
		return m.GetCertificate(clientHello)
	}
	s.htps.TLSConfig = &tls.Config{GetCertificate: getcertfn}
	return m
}

func (s *sServer) Run() error {
	s.Init()
	errc := make(chan error, 3)
	ctx, cancelf := context.WithCancel(context.Background())

	s.htps = &http.Server{
		Addr: s.Cfg.ListenAddrTLS,
	}
	autocertManager := s.updateCertificates()

	s.htps.Handler = s

	go func() {
		if autocertManager == nil {
			s.Logger.Println("Listening HTTP")
			errc <- http.ListenAndServe(s.Cfg.ListenAddr, s)
		} else {
			s.Logger.Println("Listening accepting HTTP requests to the SNI challenge")
			errc <- http.ListenAndServe(s.Cfg.ListenAddr, autocertManager.HTTPHandler(s))
		}
	}()

	certs := make([]util.Certificate, 0, len(s.domains))
	for _, v := range s.domains {
		if v.Certificate.CertFile != "" {
			certs = append(certs, v.Certificate)
		}
	}
	var wrapper *util.ServerWrapper
	if !s.Cfg.DisableTLS {
		go func() {
			if s.Cfg.Graceful {
				s.Logger.Println("Listening HTTPS (Graceful)")
				wrapper = util.NewGracefulServer(manners.NewWithServer(s.htps))
			} else {
				s.Logger.Println("Listening HTTPS (Vanilla)")
				wrapper = util.NewVanillaServer(s.htps)
			}
			errc <- util.ListenAndServeTLSSNI(wrapper, certs)
		}()
	}
	//
	// API
	if err := s.startAPI(ctx); err != nil {
		s.Logger.Println("START API ERROR:", err.Error())
	}
	//
	go func() {
		s.closeChan = make(chan os.Signal, 1)
		<-s.closeChan
		cancelf()
		if wrapper != nil {
			wrapper.Close()
		}
		errc <- nil
	}()
	//
	err := <-errc
	return err
}

func (s *sServer) Close() {
	s.closeChan <- os.Interrupt
}

func (s *sServer) Init() {
	// first start setting the number of cpu cores to use
	ncpu := runtime.NumCPU()
	if s.Cfg.NumCPU > 0 && s.Cfg.NumCPU < ncpu {
		ncpu = s.Cfg.NumCPU
	}
	runtime.GOMAXPROCS(ncpu)
}

func (s *sServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := r.Host
	if s.Cfg.Debug {
		if ho := r.Header.Get("X-Sandpiper-Host"); ho != "" {
			h = ho
		}
		s.Logger.Println("Host: " + h)
	}
	res := s.trieDomains.Find(h)
	if res == nil {
		if len(s.Cfg.FallbackDomain) > 0 {
			if s.Cfg.Debug {
				s.Logger.Println("FALLBACK DOMAIN", s.Cfg.FallbackDomain)
			}
			dom := s.domains[s.Cfg.FallbackDomain]
			if dom != nil {
				if dom.AuthMode != "" {
					switch dom.AuthMode {
					case "apikey":
						if dom.AuthValue != r.Header.Get(dom.AuthKey) {
							w.WriteHeader(http.StatusUnauthorized)
							return
						}
					}
				}
				dom.ReverseProxy(w, r)
				return
			}
		} else {
			if s.Cfg.Debug {
				s.Logger.Println("DOMAIN NOT FOUND")
			}
			http.Error(w, "domain not found "+h, http.StatusInternalServerError)
			return
		}
	}
	if res.EndRoute == nil {
		if s.Cfg.Debug {
			s.Logger.Println("ROUTE IS NULL")
		}
		http.Error(w, "route is null", http.StatusInternalServerError)
		return
	}
	if res.EndRoute.AuthMode != "" {
		switch res.EndRoute.AuthMode {
		case "apikey":
			if res.EndRoute.AuthValue != r.Header.Get(res.EndRoute.AuthKey) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
	}
	res.EndRoute.ReverseProxy(w, r)
}
