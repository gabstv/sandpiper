package route

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/gabstv/sandpiper/pkg/util"
)

type LoadBalancerConfig struct {
	//HeathCheck *HealthCheckConfig `json:"health_check,omitempty"` // TODO
	Targets []LoadBalancerTargetCfg `json:"targets" yaml:"targets"`
}

type HealthCheckConfig struct {
	// Delay in seconds
	Delay int    `json:"delay"`
	Path  string `json:"path"`
}

type LoadBalancerTargetCfg struct {
	Path string `json:"path" yaml:"path"`
}

type loadBalancer struct {
	sync.Mutex
	//HealthCheck *HealthCheckConfig // TODO
	Targets         []*loadBalancerTarget
	LastTargetIndex int // TEMP
}

type loadBalancerTarget struct {
	Count       int
	Path        string
	HealthScore int
	Proxy       *util.ReverseProxy
}

func (rs *loadBalancerTarget) URL() *url.URL {
	uri := url.URL{}
	//if rs.OutConnType == HTTP {
	uri.Scheme = "http"
	//} else {
	//	uri.Scheme = "https"
	//}
	uri.Host = rs.Path
	return &uri
}

// ServeHTTP very simple atm
func (lb *loadBalancer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var proxy *util.ReverseProxy
	lb.Lock()
	index := lb.LastTargetIndex
	lb.LastTargetIndex++
	if len(lb.Targets) <= lb.LastTargetIndex {
		lb.LastTargetIndex = 0
	}
	proxy = lb.Targets[index].Proxy
	lb.Unlock()
	proxy.ServeHTTP(rw, req)
}
