package server

import (
	"fmt"
	"github.com/gabstv/sandpiper/pathtree"
	"github.com/gabstv/sandpiper/route"
	"github.com/gabstv/sandpiper/util"
	"net/http"
	"runtime"
)

type Server struct {
	Cfg         Config
	trieDomains *pathtree.Trie
	domains     map[string]*route.Route
}

func Default() *Server {
	s := &Server{}
	s.trieDomains = pathtree.NewTrie(".")
	s.domains = make(map[string]*route.Route, 0)
	return s
}

func (s *Server) Add(r route.Route) error {
	rr := &route.Route{}
	*rr = r
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

	sv := &http.Server{
		Addr: s.Cfg.ListenAddrTLS,
	}
	certs := make([]util.Certificate, 0, len(s.domains))
	for _, v := range s.domains {
		if v.Certificate.CertFile != "" {
			certs = append(certs, v.Certificate)
		}
	}
	go func() {
		print("Running on " + sv.Addr + "\n")
		fmt.Println(certs)
		errc <- util.ListenAndServeTLSSNI(sv, certs)
	}()
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
	fmt.Println("SERVE HTTP :)")
	h := r.Host
	if s.Cfg.Debug {
		if ho := r.Header.Get("X-Sandpiper-Host"); ho != "" {
			h = ho
		}
	}
	print("H: " + h + "\n")
	res := s.trieDomains.Find(h)
	if res == nil {
		http.Error(w, "domain not found", http.StatusInternalServerError)
		return
	}
	if res.EndRoute == nil {
		http.Error(w, "route is null", http.StatusInternalServerError)
		return
	}
	res.EndRoute.ReverseProxy(w, r)
}
