package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gabstv/freeport"
	"github.com/gabstv/sandpiper/api"
	"github.com/gabstv/sandpiper/internal/pkg/route"
	"github.com/gin-gonic/gin"
)

func runAPIV1(ctx context.Context, sv Server, listen, key string, debug bool) {
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	r.GET("/health-check", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "sandpiper api")
	})

	g := r.Group("/v1")
	// authorization middleware
	g.Use(func(c *gin.Context) {
		if c.GetHeader("X-API-KEY") != key {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "invalid X-API-KEY",
			})
			return
		}
		c.Next()
	})

	//PING
	g.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"pong": time.Now(),
		})
	})

	g.GET("/routes", func(c *gin.Context) {
		vv := sv.Routes()
		c.JSON(http.StatusOK, vv)
	})

	g.GET("/fallback-domain", func(c *gin.Context) {
		cfg := sv.GetConfig()
		c.JSON(http.StatusOK, gin.H{
			"value": cfg.FallbackDomain,
		})
	})

	g.POST("/fallback-domain", func(c *gin.Context) {
		jd := struct {
			Value string `json:"value"`
		}{}
		if err := c.BindJSON(&jd); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   "json parse error: " + err.Error(),
			})
			return
		}
		routes := sv.Routes()
		if _, ok := routes[jd.Value]; !ok {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   "domain does not exist",
			})
			return
		}
		cfg := sv.GetConfig()
		cfg.FallbackDomain = jd.Value
		sv.SetConfig(cfg)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
		})
	})

	// UPSERT a new route!
	g.PUT("/route", func(c *gin.Context) {
		jd := &api.NewRoute{}
		if err := c.BindJSON(jd); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		rrr := route.Route{
			Domain:   jd.Domain,
			Autocert: jd.Autocert,
			Server: route.RouteServer{
				OutAddress: jd.OutPath,
			},
		}

		newport := 0

		switch jd.OutType {
		case "http", "HTTP":
			sv := rrr.Server
			sv.OutConnType = route.HTTP
			rrr.Server = sv
		case "https_skip_verify", "HTTP_SKIP_VERIFY":
			sv := rrr.Server
			sv.OutConnType = route.HTTPS_SKIP_VERIFY
			rrr.Server = sv
		case "https", "HTTPS":
			sv := rrr.Server
			sv.OutConnType = route.HTTPS_VERIFY
			rrr.Server = sv
		case "auto", "AUTO":
			ntcp, err := freeport.TCP()
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"error":   "tcp port err: " + err.Error(),
				})
				return
			}
			sv2 := rrr.Server
			if sv2.OutAddress == "" {
				sv2.OutAddress = fmt.Sprintf("localhost:%d", ntcp)
			} else {
				sv2.OutAddress = fmt.Sprintf("%v:%d", sv2.OutAddress, ntcp)
			}
			rrr.Server = sv2
			newport = ntcp
		case "redirect", "REDIRECT":
			sv := rrr.Server
			sv.OutConnType = route.REDIRECT
			rrr.Server = sv
		}

		if err := sv.Add(rrr); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		if newport > 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"port":    newport,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
			})
		}
	})

	srv := &http.Server{
		Addr:    listen,
		Handler: r,
	}
	go srv.ListenAndServe()
	for ctx.Err() == nil {
		// do nothing
		select {
		case <-ctx.Done():
			break
		case <-time.After(time.Second * 5):
		}
	}
	srv.Shutdown(ctx)
}
