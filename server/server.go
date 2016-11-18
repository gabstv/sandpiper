package server

import (
	"crypto/tls"
	"github.com/gabstv/sandpiper/pathtree"
	"github.com/gabstv/sandpiper/route"
	"github.com/gabstv/sandpiper/util"
	"golang.org/x/crypto/acme/autocert"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

type Server struct {
	Cfg         Config
	trieDomains *pathtree.Trie
	domains     map[string]*route.Route
	Logger      *log.Logger
}

func Default() *Server {
	s := &Server{}
	s.trieDomains = pathtree.NewTrie(".")
	s.domains = make(map[string]*route.Route, 0)
	s.Logger = log.New(os.Stderr, "[sp server] ", log.LstdFlags)
	return s
}

func (s *Server) DebugLog() {

}

func (s *Server) Add(r route.Route) error {
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
	return nil
}

func (s *Server) Run() error {
	s.Init()
	errc := make(chan error, 3)
	go func() {
		errc <- http.ListenAndServe(s.Cfg.ListenAddr, s)
	}()

	// Autocert
	autocdomains := make([]string, 0)
	for _, v := range s.domains {
		if v.Autocert {
			autocdomains = append(autocdomains, v.Domain)
		}
	}
	var sv *http.Server
	if len(autocdomains) > 0 {
		m := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(autocdomains...),
		}
		sv = &http.Server{
			Addr:      s.Cfg.ListenAddrTLS,
			TLSConfig: &tls.Config{GetCertificate: m.GetCertificate},
		}
	} else {
		sv = &http.Server{
			Addr: s.Cfg.ListenAddrTLS,
		}
	}

	sv.Handler = s
	certs := make([]util.Certificate, 0, len(s.domains))
	for _, v := range s.domains {
		if v.Certificate.CertFile != "" {
			certs = append(certs, v.Certificate)
		}
	}
	if len(certs) > 0 || len(autocdomains) > 0 {
		go func() {
			errc <- util.ListenAndServeTLSSNI(sv, certs)
		}()
	}
	err := <-errc
	return err
}

func (s *Server) Init() {
	// first start setting the number of cpu cores to use
	ncpu := runtime.NumCPU()
	if s.Cfg.NumCPU > 0 && s.Cfg.NumCPU < ncpu {
		ncpu = s.Cfg.NumCPU
	}
	runtime.GOMAXPROCS(ncpu)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	res.EndRoute.ReverseProxy(w, r)
}
