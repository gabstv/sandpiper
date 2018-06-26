package route

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gabstv/sandpiper/util"
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
)

type Route struct {
	Domain      string           `json:"domain"`
	Server      RouteServer      `json:"server"` //TODO: maybe support load balancing in the future
	Certificate util.Certificate `json:"certificate"`
	Autocert    bool             `json:"autocert"`
	WsCFG       util.WsConfig    `json:"wscfg"`
	fn          func(w http.ResponseWriter, r *http.Request)
	AuthMode    string `json:"auth_mode"`
	AuthKey     string `json:"auth_key"`
	AuthValue   string `json:"auth_value"`
	ForceHTTPS  bool   `json:"force_https"`
}

type RouteServer struct {
	OutConnType ConnType `json:"out_conn_type"`
	OutAddress  string   `json:"out_address"`
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
		} else {
			rp := buildReverseProxy(rt)
			if rt.ForceHTTPS {
				rt.fn = func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("X-Forwarded-Proto") == "http" {
						url2 := *r.URL
						url2.Scheme = "https"
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
	rp := util.NewSingleHostReverseProxy(rt.Server.URL(), rt.WsCFG)
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
