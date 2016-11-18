package route

import (
	"crypto/tls"
	"github.com/gabstv/sandpiper/util"
	"net"
	"net/http"
	//"net/http/httputil"
	"net/url"
	"time"
)

type ConnType int

const (
	HTTP              ConnType = 0
	HTTPS_VERIFY      ConnType = 1
	HTTPS_SKIP_VERIFY ConnType = 2
)

type Route struct {
	Domain      string
	Server      RouteServer //TODO: maybe support load balancing in the future
	Certificate util.Certificate
	Autocert    bool
	WsCFG       util.WsConfig
	rp          *util.ReverseProxy
}

type RouteServer struct {
	OutConnType ConnType
	OutAddress  string
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

func (rt *Route) ReverseProxy(w http.ResponseWriter, r *http.Request) {
	if rt.rp == nil {
		rt.rp = util.NewSingleHostReverseProxy(rt.Server.URL(), rt.WsCFG)
		if rt.Server.OutConnType == HTTPS_SKIP_VERIFY {
			rt.rp.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				Dial: func(network, addr string) (net.Conn, error) {
					return net.DialTimeout(network, addr, time.Duration(60*time.Second))
				},
			}
		}
	}
	rt.rp.ServeHTTP(w, r)
}
