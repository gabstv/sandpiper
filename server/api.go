package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func runAPIV1(ctx context.Context, sv Server, listen, key string, debug bool) {
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	g := r.Group("/v1")
	// authorization middleware
	g.Use(func(c *gin.Context) {
		if c.GetHeader("X_API_KEY") != key {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "invalid X_API_KEY",
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

	// UPSERT a new route!
	//g.POST("/route", func(c *gin.Context){
	//
	//})

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
