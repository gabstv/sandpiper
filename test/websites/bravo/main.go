package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/health_check", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "github.com/gabstv/sandpiper/test/websites/bravo")
	})
	r.GET("/shutdown", func(c *gin.Context) {
		c.String(http.StatusOK, "bravo shutdown")
		time.Sleep(time.Millisecond * 50)
		os.Exit(0)
	})
	if err := r.Run(":9002"); err != nil {
		fmt.Println(err.Error())
	}
}
