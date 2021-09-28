package main

import (
	"net/http"

	"github.com/gabstv/sandpiper/internal/pkg/route"
	"github.com/gabstv/sandpiper/pkg/server"
	"github.com/gabstv/sandpiper/pkg/util"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := server.Config{
		Debug:          true,
		ListenAddr:     ":80",
		ListenAddrTLS:  ":443",
		FallbackDomain: "latest.happytanks.com",
		LetsEncryptURL: "https://acme-staging.api.letsencrypt.org/directory",
	}
	sps := server.Default(&cfg)
	sps.Add(route.Route{
		Domain: "latest.happytanks.com",
		Server: route.RouteServer{
			OutAddress:  "localhost:9406",
			OutConnType: route.HTTP,
		},
		WsCFG: util.WsConfig{
			Enabled:             true,
			ReadBufferSize:      1024,
			WriteBufferSize:     1024,
			ReadDeadlineSeconds: 60,
		},
		Autocert: true,
	})
	//acme.LetsEncryptURL

	gr := gin.Default()

	gr.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "web loaded ok")
	})

	go func() {
		gr.Run(":9406")
	}()
	sps.Run()
}
