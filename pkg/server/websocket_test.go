package server

import (
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gabstv/sandpiper/pkg/util"
	"github.com/gabstv/sandpiper/route"
	"github.com/gorilla/websocket"
)

type wsTestServer struct {
	upgrader websocket.Upgrader
	t        *testing.T
	send     chan []byte
	ws       *websocket.Conn
	Numm     int
}

func (s *wsTestServer) wswrite() {
	ticker := time.NewTicker(time.Second * 50)
	defer func() {
		ticker.Stop()
		s.ws.Close()
	}()
	for {
		select {
		case msg, ok := <-s.send:
			if !ok {
				s.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := s.write(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := s.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (s *wsTestServer) wsread() {
	defer func() {
		s.ws.Close()
	}()
	s.ws.SetReadLimit(512)
	s.ws.SetReadDeadline(time.Now().Add(time.Second * 60))
	s.ws.SetPongHandler(func(string) error { s.ws.SetReadDeadline(time.Now().Add(time.Second * 60)); return nil })
	for {
		_, msg, err := s.ws.ReadMessage()
		if err != nil {
			return // probably EOF
		}
		s.t.Logf("Received '%v'", string(msg))
		s.Numm++
	}
}

// write writes a message with the given message type and payload.
func (s *wsTestServer) write(mt int, payload []byte) error {
	s.ws.SetWriteDeadline(time.Now().Add(time.Second * 10))
	return s.ws.WriteMessage(mt, payload)
}

func (t *wsTestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.t.Logf("Headers %v", r.Header)
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	ws, err := t.upgrader.Upgrade(w, r, nil)
	if err != nil {
		t.t.Fatalf("Websocket upgrader error! %s", err.Error())
	}
	t.ws = ws
	go t.wswrite()
	t.wsread()
}

func TestWebsocket(t *testing.T) {
	s0 := &wsTestServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		t:    t,
		send: make(chan []byte, 256),
	}
	go func() {
		err := http.ListenAndServe(":9099", s0)
		if err != nil {
			t.Fatal(err)
		}
	}()
	//
	sv := Default(&Config{
		Debug:         true,
		ListenAddr:    ":9100",
		ListenAddrTLS: ":9101",
	})
	r0 := route.Route{
		Domain: "example.com",
		Server: route.RouteServer{
			OutConnType: route.HTTP,
			OutAddress:  "localhost:9099",
		},
		WsCFG: util.WsConfig{Enabled: true},
	}
	err := sv.Add(r0)
	if err != nil {
		t.Fatal(err)
	}

	uri, _ := url.Parse("ws://localhost:9100")
	h := http.Header{}
	h.Set("X-Sandpiper-Host", "example.com")
	//h.Set("Upgrade", "websocket")

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

	ws, _, err := websocket.NewClient(c, uri, h, 1024, 1024)
	if err != nil {
		t.Fatalf("Could not connect %s", err)
	}
	if e0 := ws.WriteMessage(websocket.TextMessage, []byte("hello 1!")); e0 != nil {
		t.Fatalf("ws.WriteMessage %s", e0)
	}
	time.Sleep(time.Second * 1)
	if e0 := ws.WriteMessage(websocket.TextMessage, []byte("hello 2!")); e0 != nil {
		t.Fatalf("ws.WriteMessage %s", e0)
	}
	time.Sleep(time.Second * 7)
	if e0 := ws.WriteMessage(websocket.TextMessage, []byte("hello 3!")); e0 != nil {
		t.Fatalf("ws.WriteMessage %s", e0)
	}
	time.Sleep(time.Second * 1)
	ws.WriteMessage(websocket.CloseMessage, []byte{})
	time.Sleep(time.Second * 1)
	if s0.Numm != 3 {
		t.Fatalf("Should have received 3 messages!")
	}
}
