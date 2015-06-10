package util

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
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

func ListenAndServeTLSSNI(server *http.Server, certs []Certificate) error {
	addr := server.Addr
	if addr == "" {
		addr = ":https"
	}
	config := &tls.Config{}
	if server.TLSConfig != nil {
		*config = *server.TLSConfig
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
	return server.Serve(tlsl)
}
