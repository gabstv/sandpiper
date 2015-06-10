package server

import (
	"github.com/gabstv/sandpiper/route"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
)

type wsTestServer struct {
	upgrader websocket.Upgrader
	t        *testing.T
}

func (t *wsTestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.t.Logf("Headers %v", r.Header)
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	_, err := t.upgrader.Upgrade(w, r, nil)
	if err != nil {
		t.t.Fatalf("Websocket upgrader error! %s", err.Error())
	}
}

func TestWebsocket(t *testing.T) {
	s0 := &wsTestServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		t: t,
	}
	go func() {
		err := http.ListenAndServe(":9099", s0)
		if err != nil {
			t.Fatal(err)
		}
	}()
	//
	sv := Default()
	sv.Cfg.Debug = true
	sv.Cfg.ListenAddr = ":9100"
	sv.Cfg.ListenAddrTLS = ":9101"
	r0 := route.Route{
		Domain: "example.com",
		Server: route.RouteServer{
			OutConnType: route.HTTP,
			OutAddress:  "localhost:9099",
		},
	}
	err := sv.Add(r0)
	if err != nil {
		t.Fatal(err)
	}

	uri, _ := url.Parse("http://localhost:9100")
	h := http.Header{}
	h.Set("X-Sandpiper-Host", "example.com")
	h.Set("Upgrade", "websocket")

	go func() {
		err := sv.Run()
		if err != nil {
			t.Fatalf("sv.Run ERR: %s", err)
		}
	}()

	time.Sleep(time.Millisecond * 100)

	c, err := net.Dial("tcp", uri.Host)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	_, _, err = websocket.NewClient(c, uri, h, 1024, 1024)
	if err != nil {
		t.Fatalf("Could not connect %s", err)
	}
}
