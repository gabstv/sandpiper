package util

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gabstv/manners"
)

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

type Certificate struct {
	CertFile string
	KeyFile  string
}

type ServerWrapper struct {
	vanilla  *http.Server
	graceful *manners.GracefulServer
}

func newServerWrapper(vanilla *http.Server, graceful *manners.GracefulServer) *ServerWrapper {
	return &ServerWrapper{
		vanilla:  vanilla,
		graceful: graceful,
	}
}

func NewVanillaServer(vanilla *http.Server) *ServerWrapper {
	return newServerWrapper(vanilla, nil)
}

func NewGracefulServer(graceful *manners.GracefulServer) *ServerWrapper {
	return newServerWrapper(nil, graceful)
}

func (w *ServerWrapper) GetAddr() string {
	if w.vanilla != nil {
		return w.vanilla.Addr
	}
	return w.graceful.Addr
}

func (w *ServerWrapper) GetTLSConfig() *tls.Config {
	if w.vanilla != nil {
		return w.vanilla.TLSConfig
	}
	return w.graceful.TLSConfig
}

func (w *ServerWrapper) Serve(l net.Listener) error {
	if w.vanilla != nil {
		return w.vanilla.Serve(l)
	}
	return w.graceful.Serve(l)
}

func (w *ServerWrapper) IsGraceful() bool {
	if w.vanilla != nil {
		return false
	}
	return true
}

func (w *ServerWrapper) Close() bool {
	if w.graceful != nil {
		log.Println("Shutting down gracefully...")
		return w.graceful.Close()
	}
	log.Println("Shutting down...")
	w.vanilla.Close()
	return true
}

func ListenAndServeTLSSNI(server *ServerWrapper, certs []Certificate) error {
	graceful := server.IsGraceful()
	addr := server.GetAddr()
	if addr == "" {
		addr = ":https"
	}
	config := &tls.Config{}
	if server.GetTLSConfig() != nil {
		cfcf := server.GetTLSConfig()
		*config = *cfcf
	}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}

	config.Certificates = make([]tls.Certificate, len(certs))
	for k, v := range certs {
		var err error
		config.Certificates[k], err = tls.LoadX509KeyPair(v.CertFile, v.KeyFile)
		if err != nil {
			return err
		}
	}

	config.BuildNameToCertificate()

	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	tlsl := tls.NewListener(tcpKeepAliveListener{conn.(*net.TCPListener)}, config)

	//TODO: test this after graceful update
	//TODO: fix malformed http resp when getting TLS
	// net/http: HTTP/1.x transport connection broken: malformed HTTP response "\x15\x03\x01\x00\x02\x02\x16"
	// add this to the docs!
	if graceful {
		graceful_l := manners.NewTLSListener(tlsl, config)
		return server.Serve(graceful_l)
	}
	return server.Serve(tlsl)
}
