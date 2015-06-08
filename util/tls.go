package util

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
)

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
	fmt.Println("NAME TO CERT", config.NameToCertificate)

	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return server.Serve(tls.NewListener(conn, config))
}
