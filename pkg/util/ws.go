package util

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// DEBUG will print extra information to stdout
var DEBUG = false

func dlogln(stuff ...interface{}) {
	if DEBUG {
		log.Println(stuff...)
	}
}

// ReverseProxy is an HTTP Handler that takes an incoming request and
// sends it to another server, proxying the response back to the
// client.
type ReverseProxy struct {
	// Director must be a function which modifies
	// the request into a new request to be sent
	// using Transport. Its response is then copied
	// back to the original client unmodified.
	Director func(*http.Request)

	// The transport used to perform proxy requests.
	// If nil, http.DefaultTransport is used.
	Transport http.RoundTripper

	// FlushInterval specifies the flush interval
	// to flush to the client while copying the
	// response body.
	// If zero, no periodic flushing is done.
	FlushInterval time.Duration

	// ErrorLog specifies an optional logger for errors
	// that occur when attempting to proxy the request.
	// If nil, logging goes to os.Stderr via the log package's
	// standard logger.
	ErrorLog *log.Logger

	// Configure Websocket
	WsCFG WsConfig
}

// WsConfig websockets configuration
type WsConfig struct {
	Enabled             bool          `yaml:"enabled"`
	ReadBufferSize      int           `yaml:"read_buffer_size"`
	WriteBufferSize     int           `yaml:"write_buffer_size"`
	ReadDeadlineSeconds time.Duration `yaml:"read_deadline_seconds"`
}

type wsbridge struct {
	proxy2endpoint *websocket.Conn
	client2proxy   *websocket.Conn
	rp             *ReverseProxy
}

func (b *wsbridge) EndpointLoopRead() {
	defer func() {
		//ticker.Stop()
		if b.proxy2endpoint != nil {
			b.proxy2endpoint.Close()
		}
		if b.client2proxy != nil {
			b.client2proxy.Close()
		}
	}()
	if b.proxy2endpoint == nil {
		return
	}
	b.proxy2endpoint.SetReadLimit(int64(b.rp.WsCFG.ReadBufferSize))
	b.proxy2endpoint.SetReadDeadline(time.Now().Add(b.rp.WsCFG.ReadDeadlineSeconds))
	b.proxy2endpoint.SetPongHandler(func(string) error {
		b.proxy2endpoint.SetReadDeadline(time.Now().Add(b.rp.WsCFG.ReadDeadlineSeconds))
		//TODO: ping the endpoint
		return nil
	})
	for {
		if b.proxy2endpoint == nil {
			return
		}
		b.proxy2endpoint.SetReadDeadline(time.Now().Add(b.rp.WsCFG.ReadDeadlineSeconds))
		b.proxy2endpoint.SetWriteDeadline(time.Now().Add(b.rp.WsCFG.ReadDeadlineSeconds))
		mtype, rdr, err := b.proxy2endpoint.NextReader()
		if err != nil {
			return
		}
		if b.client2proxy == nil {
			return
		}
		wc, err := b.client2proxy.NextWriter(mtype)
		if err != nil {
			return
		}
		io.Copy(wc, rdr)
		wc.Close()
		//TODO: react on close
	}
}

func (b *wsbridge) ClientLoopRead() {
	//ticker := time.NewTicker(time.Second * 50)
	defer func() {
		//ticker.Stop()
		if b.proxy2endpoint != nil {
			b.proxy2endpoint.Close()
		}
		if b.client2proxy != nil {
			b.client2proxy.Close()
		}
	}()
	if b.client2proxy == nil {
		return
	}
	b.client2proxy.SetReadLimit(int64(b.rp.WsCFG.ReadBufferSize))
	b.client2proxy.SetReadDeadline(time.Now().Add(b.rp.WsCFG.ReadDeadlineSeconds))
	b.client2proxy.SetPongHandler(func(string) error {
		b.client2proxy.SetReadDeadline(time.Now().Add(b.rp.WsCFG.ReadDeadlineSeconds))
		//TODO: ping the endpoint
		return nil
	})
	for {
		if b.client2proxy == nil {
			return
		}
		b.client2proxy.SetReadDeadline(time.Now().Add(b.rp.WsCFG.ReadDeadlineSeconds))
		b.client2proxy.SetWriteDeadline(time.Now().Add(b.rp.WsCFG.ReadDeadlineSeconds))
		mtype, rdr, err := b.client2proxy.NextReader()
		if err != nil {
			return
		}
		if b.proxy2endpoint == nil {
			return
		}
		wc, err := b.proxy2endpoint.NextWriter(mtype)
		if err != nil {
			return
		}
		io.Copy(wc, rdr)
		wc.Close()
		//TODO: react on close
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

// NewSingleHostReverseProxy returns a new ReverseProxy that rewrites
// URLs to the scheme, host, and base path provided in target. If the
// target's path is "/base" and the incoming request was for "/dir",
// the target request will be for /base/dir.
func NewSingleHostReverseProxy(target *url.URL, wsconfig WsConfig) *ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
	}
	return &ReverseProxy{Director: director, WsCFG: wsconfig}
}

func (p *ReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	transport := p.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	outreq := new(http.Request)
	*outreq = *req // includes shallow copies of maps, but okay

	p.Director(outreq)
	outreq.Proto = "HTTP/1.1"
	outreq.ProtoMajor = 1
	outreq.ProtoMinor = 1
	outreq.Close = false

	// support for Websockets
	useWebsockets := false
	if p.WsCFG.Enabled {
		if v1 := req.Header.Get("Upgrade"); v1 == "websocket" || v1 == "Websocket" {
			if v0 := strings.ToLower(req.Header.Get("Connection")); strings.Contains(v0, "upgrade") {
				if req.Method != "GET" {
					// cut the cord earlier to avoid useless cpu use
					http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				useWebsockets = true
			}
		}
	}

	// Remove hop-by-hop headers to the backend.  Especially
	// important is "Connection" because we want a persistent
	// connection, regardless of what the client sent to us.  This
	// is modifying the same underlying map from req (shallow
	// copied above) so we only copy it if necessary.
	copiedHeaders := false
	for _, h := range hopHeaders {
		if outreq.Header.Get(h) != "" {
			if !copiedHeaders {
				outreq.Header = make(http.Header)
				copyHeader(outreq.Header, req.Header)
				copiedHeaders = true
			}
			outreq.Header.Del(h)
		}
	}

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		if prior, ok := outreq.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		outreq.Header.Set("X-Forwarded-For", clientIP)
	}
	// pass the origin al protocol
	aa := ""
	if req.TLS != nil { // TODO: fix url scheme
		aa = "s"
	}
	outreq.Header.Set("X-Forwarded-Proto", req.URL.Scheme+aa)
	outreq.Header.Set("X-Forwarded-Host", req.Host)

	if useWebsockets {
		// connect to the proxied server and asks for websockets!
		c, err := net.Dial("tcp", outreq.URL.Host)
		if err != nil {
			dlogln("net dial tcp error", err)
			http.Error(rw, "Internal Server Error - "+err.Error(), http.StatusInternalServerError)
			return
		}
		url2 := *outreq.URL
		url2.Scheme = "ws"
		outreq.Header.Set("X-Forwarded-Proto", url2.Scheme+aa)

		outreq.Header.Del("Sec-Websocket-Key")
		outreq.Header.Del("Sec-Websocket-Version")
		outreq.Header.Del("Sec-Websocket-Extensions")

		proxy2endserver, _, err := websocket.NewClient(c, &url2, outreq.Header, p.WsCFG.ReadBufferSize, p.WsCFG.WriteBufferSize)
		if err != nil {
			dlogln("websocket newclient", err, url2.String(), outreq.Header)
			http.Error(rw, "Internal Server Error - "+err.Error(), http.StatusInternalServerError)
			return
		}

		upgrader := websocket.Upgrader{
			ReadBufferSize:  p.WsCFG.ReadBufferSize,
			WriteBufferSize: p.WsCFG.WriteBufferSize,
		}
		//req.Header.Set("Connection", "Upgrade")
		//req.Header.Set("Upgrade", "websocket")
		client2proxy, err := upgrader.Upgrade(rw, req, nil)
		if err != nil {
			dlogln("upgrader error", err)
			http.Error(rw, "Internal Server Error - "+err.Error(), http.StatusInternalServerError)
		}
		//
		wsb := &wsbridge{
			proxy2endpoint: proxy2endserver,
			client2proxy:   client2proxy,
			rp:             p,
		}
		go wsb.ClientLoopRead()
		wsb.EndpointLoopRead()
		//
		return
	}

	res, err := transport.RoundTrip(outreq)
	if err != nil {
		p.logf("http: proxy error: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	for _, h := range hopHeaders {
		res.Header.Del(h)
	}

	copyHeader(rw.Header(), res.Header)

	rw.WriteHeader(res.StatusCode)
	p.copyResponse(rw, res.Body)
}

func (p *ReverseProxy) copyResponse(dst io.Writer, src io.Reader) {
	if p.FlushInterval != 0 {
		if wf, ok := dst.(writeFlusher); ok {
			mlw := &maxLatencyWriter{
				dst:     wf,
				latency: p.FlushInterval,
				done:    make(chan bool),
			}
			go mlw.flushLoop()
			defer mlw.stop()
			dst = mlw
		}
	}

	io.Copy(dst, src)
}

func (p *ReverseProxy) logf(format string, args ...interface{}) {
	if p.ErrorLog != nil {
		p.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

type writeFlusher interface {
	io.Writer
	http.Flusher
}

type maxLatencyWriter struct {
	dst     writeFlusher
	latency time.Duration

	lk   sync.Mutex // protects Write + Flush
	done chan bool
}

func (m *maxLatencyWriter) Write(p []byte) (int, error) {
	m.lk.Lock()
	defer m.lk.Unlock()
	return m.dst.Write(p)
}

func (m *maxLatencyWriter) flushLoop() {
	t := time.NewTicker(m.latency)
	defer t.Stop()
	for {
		select {
		case <-m.done:
			if onExitFlushLoop != nil {
				onExitFlushLoop()
			}
			return
		case <-t.C:
			m.lk.Lock()
			m.dst.Flush()
			m.lk.Unlock()
		}
	}
}

func (m *maxLatencyWriter) stop() { m.done <- true }

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// onExitFlushLoop is a callback set by tests to detect the state of the
// flushLoop() goroutine.
var onExitFlushLoop func()
