package route

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gabstv/sandpiper/pkg/util"
)

// ConnType specifies the connection type
type ConnType int

const (
	// HTTP - Connect to an endpoint with the http protocol
	HTTP ConnType = 0
	// HTTPS_VERIFY - Connect to an endpoint with the https protocol
	HTTPS_VERIFY ConnType = 1
	// HTTPS_SKIP_VERIFY - Connect to an endpoint with the https protocol (without verifying the server identity)
	HTTPS_SKIP_VERIFY ConnType = 2
	// REDIRECT - Redirect to the provided address
	REDIRECT ConnType = 3
	// LOAD_BALANCER - Load balancer mode
	LOAD_BALANCER ConnType = 4
)

func ParseConnType(v string) ConnType {
	switch strings.ToUpper(v) {
	case "HTTP", "0":
		return HTTP
	case "HTTPS_VERIFY", "HTTPS", "1":
		return HTTPS_VERIFY
	case "HTTPS_SKIP_VERIFY", "2":
		return HTTPS_SKIP_VERIFY
	case "REDIRECT", "3":
		return REDIRECT
	case "LOAD_BALANCER", "4":
		return LOAD_BALANCER
	}
	return HTTP
}

var defaultWebsocks = util.WsConfig{
	Enabled:             true,
	ReadBufferSize:      4096,
	WriteBufferSize:     4096,
	ReadDeadlineSeconds: time.Second * 60,
}

type Route struct {
	Domain        string           `json:"domain" yaml:"domain"`
	Server        RouteServer      `json:"server" yaml:"server"`
	Certificate   util.Certificate `json:"certificate" yaml:"certificate"`
	Autocert      bool             `json:"autocert" yaml:"autocert"`
	WsCFG         util.WsConfig    `json:"wscfg" yaml:"wscfg"`
	fn            func(w http.ResponseWriter, r *http.Request)
	AuthMode      string `json:"auth_mode" yaml:"auth_mode"`
	AuthKey       string `json:"auth_key" yaml:"auth_key"`
	AuthValue     string `json:"auth_value" yaml:"auth_value"`
	ForceHTTPS    bool   `json:"force_https" yaml:"force_https"`
	FlushInterval int    `json:"flush_interval" yaml:"flush_interval"`
}

type RouteServer struct {
	OutConnType  ConnType            `json:"out_conn_type" yaml:"out_conn_type"`
	OutAddress   string              `json:"out_address,omitempty" yaml:"out_address"`
	LoadBalancer *LoadBalancerConfig `json:"load_balancer,omitempty" yaml:"load_balancer"`
}

func (rs *RouteServer) URL() *url.URL {
	uri := url.URL{}
	if rs.OutConnType == HTTP {
		uri.Scheme = "http"
	} else {
		uri.Scheme = "https"
	}
	uri.Host = rs.OutAddress
	return &uri
}

// ReverseProxy will route all requests for this route configuration
func (rt *Route) ReverseProxy(w http.ResponseWriter, r *http.Request) {
	if rt.fn == nil {
		if rt.Server.OutConnType == REDIRECT {
			base, err := url.Parse(rt.Server.OutAddress)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Could not redirect (invalid URL); " + err.Error()))
				return
			}
			rt.fn = func(w http.ResponseWriter, r *http.Request) {
				url1, err := url.Parse(r.URL.Path)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Could not redirect (invalid path); " + err.Error()))
					return
				}
				url2 := base.ResolveReference(url1)
				http.Redirect(w, r, url2.String(), http.StatusPermanentRedirect)
			}
		} else if rt.Server.OutConnType == LOAD_BALANCER {
			if rt.Server.LoadBalancer == nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("could not serve (load balancer configuration is nil)"))
				return
			}
			lblb := &loadBalancer{}
			lblb.Targets = make([]*loadBalancerTarget, 0)
			for _, v := range rt.Server.LoadBalancer.Targets {
				lbt := &loadBalancerTarget{
					Path:        v.Path,
					HealthScore: 100,
					Count:       0,
				}
				rp := util.NewSingleHostReverseProxy(lbt.URL(), defaultWebsocks, time.Second)
				lbt.Proxy = rp
				lblb.Targets = append(lblb.Targets, lbt)
			}
			rt.fn = func(w http.ResponseWriter, r *http.Request) {
				lblb.ServeHTTP(w, r)
			}
		} else {
			rp := buildReverseProxy(rt)
			if rt.ForceHTTPS {
				rt.fn = func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("X-Forwarded-Proto") == "http" {
						url2 := *r.URL
						url2.Scheme = "https"
						url2.Host = r.Host
						http.Redirect(w, r, url2.String(), http.StatusPermanentRedirect)
						return
					}
					rp.ServeHTTP(w, r)
				}
			} else {
				rt.fn = func(w http.ResponseWriter, r *http.Request) {
					rp.ServeHTTP(w, r)
				}
			}
		}
	}
	rt.fn(w, r)
}

func buildReverseProxy(rt *Route) *util.ReverseProxy {
	rp := util.NewSingleHostReverseProxy(rt.Server.URL(), rt.WsCFG, time.Duration(rt.FlushInterval)*time.Second)
	if rt.Server.OutConnType == HTTPS_SKIP_VERIFY {
		rp.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Dial: func(network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, time.Duration(60*time.Second))
			},
		}
	}
	return rp
}
