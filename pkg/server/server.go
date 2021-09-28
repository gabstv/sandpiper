package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/gabstv/sandpiper/internal/pkg/pathtree"
	"github.com/gabstv/sandpiper/internal/pkg/route"
	"github.com/gabstv/sandpiper/pkg/s3dircache"
	"github.com/gabstv/sandpiper/pkg/util"
	"github.com/pkg/errors"
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
	Routes() map[string]route.Route
	GetConfig() Config
	SetConfig(cfg Config)
}

type sServer struct {
	Cfg             Config
	trieDomains     *pathtree.Trie
	domains         map[string]*route.Route
	Logger          *log.Logger
	closeChan       chan os.Signal
	htps            *http.Server
	autocertDomains map[string]bool
}

func (s *sServer) GetConfig() Config {
	return s.Cfg
}

func (s *sServer) SetConfig(cfg Config) {
	s.Cfg = cfg
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
	s.autocertDomains = make(map[string]bool)
	return s
}

func (s *sServer) Routes() map[string]route.Route {
	mm := make(map[string]route.Route)
	for k, v := range s.domains {
		mm[k] = *v
	}
	return mm
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

	rr.SetupWsCfgDefaults()

	err := s.trieDomains.Add(r.Domain, rr)
	if err != nil {
		return err
	}
	s.domains[r.Domain] = rr
	if r.Autocert {
		s.autocertDomains[r.Domain] = true
	}
	return nil
}

func (s *sServer) autocertHostPolicy(ctx context.Context, host string) error {
	if s.autocertDomains[host] {
		return nil
	}
	if s.Cfg.AutocertAll {
		s.Logger.Println("autocertHostPolicy AutocertAll:", host)
		return nil
	}
	return fmt.Errorf("acme/autocert: host %s NOT allowed", host)
}

func (s *sServer) setupCertificates() *autocert.Manager {
	var m *autocert.Manager
	cpath := "/tmp/sandpiper"
	if s.Cfg.CachePath != "" {
		cpath = s.Cfg.CachePath
	}
	var dcache autocert.Cache
	if s.Cfg.S3Cache {
		dcache = &s3dircache.C{
			AwsID:     s.Cfg.S3ID,
			AwsSecret: s.Cfg.S3Secret,
			Region:    s.Cfg.S3Region,
			Bucket:    s.Cfg.S3Bucket,
			Folder:    s.Cfg.S3Folder,
		}
	} else {
		dc := autocert.DirCache(cpath)
		dcache = &dc
	}

	m = &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: s.autocertHostPolicy,
		Cache:      dcache,
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
		s.Logger.Println("get certificate", *clientHello)
		if dom, ok := certs[clientHello.ServerName]; ok {
			return &dom, nil
		}
		return m.GetCertificate(clientHello)
	}
	if s.Cfg.LetsEncryptURL == "dev" {
		getcertfn = func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			s.Logger.Println("get certificate (dev)", *clientHello)
			if dom, ok := certs[clientHello.ServerName]; ok {
				return &dom, nil
			}
			cccert, err := createCert(clientHello)
			if err != nil {
				return nil, err
			}
			certs[clientHello.ServerName] = cccert
			return &cccert, nil
		}
	}
	if s.htps != nil {
		s.htps.TLSConfig = m.TLSConfig() //&tls.Config{GetCertificate: getcertfn}
		s.htps.TLSConfig.GetCertificate = getcertfn
	} else {
		s.Logger.Println("s.htps WAS NIL")
		s.htps = &http.Server{
			Addr: s.Cfg.ListenAddrTLS,
		}
		s.htps.Handler = s
		s.htps.TLSConfig = m.TLSConfig() //&tls.Config{GetCertificate: getcertfn}
		s.htps.TLSConfig.GetCertificate = getcertfn
	}
	return m
}

func (s *sServer) Run() error {
	s.Init()
	errc := make(chan error, 3)
	ctx, cancelf := context.WithCancel(context.Background())

	s.htps = &http.Server{
		Addr: s.Cfg.ListenAddrTLS,
	}
	autocertManager := s.setupCertificates()

	s.htps.Handler = s

	go func() {
		if autocertManager == nil || s.Cfg.DisableTLS {
			s.Logger.Println("Listening HTTP")
			if autocertManager == nil {
				s.Logger.Println("autocertManager is nil")
			}
			lserr := http.ListenAndServe(s.Cfg.ListenAddr, s)
			if lserr != nil {
				errc <- errors.Wrapf(lserr, "[default] http.ListenAndServe(%q)", s.Cfg.ListenAddr)
			}
		} else {
			s.Logger.Println("Listening accepting HTTP requests to the SNI challenge")
			lserr := http.ListenAndServe(s.Cfg.ListenAddr, autocertManager.HTTPHandler(s))
			if lserr != nil {
				errc <- errors.Wrapf(lserr, "[default] http.ListenAndServe(%q)", s.Cfg.ListenAddr)
			}
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
				//s.Logger.Println("Listening HTTPS (Graceful)")
				//wrapper = util.NewGracefulServer(manners.NewWithServer(s.htps))
				s.Logger.Println("Graceful mode disabled due to inability to resolve autocert challenges properly")
			}
			//} else {
			s.Logger.Println("Listening HTTPS (Vanilla)")
			wrapper = util.NewVanillaServer(s.htps)
			//}
			lserr := util.ListenAndServeTLSSNI(wrapper, certs)
			if lserr != nil {
				errc <- errors.Wrap(lserr, "util.ListenAndServeTLSSNI")
			}
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
			} else {
				if s.Cfg.Debug {
					s.Logger.Println("FALLBACK DOMAIN NOT FOUND")
				}
				http.Error(w, "fallback domain not found "+h, http.StatusInternalServerError)
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
