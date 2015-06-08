package main

import (
	"github.com/gabstv/sandpiper/server"
)

func main() {
	s := server.Default()
	s.Run()
}
