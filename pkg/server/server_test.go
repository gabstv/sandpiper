package server

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gabstv/sandpiper/internal/pkg/route"
	"github.com/gabstv/sandpiper/pkg/util"
)

func testRequest(host, method, path string) (respRec *httptest.ResponseRecorder, r *http.Request) {
	r, _ = http.NewRequest(method, path, nil)
	r.Header.Set("X-Sandpiper-Host", host)
	respRec = httptest.NewRecorder()
	return
}

type testServer struct {
}

func (t *testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello!"))
}

func TestServerBasic(t *testing.T) {
	//
	// start dummy server
	s0 := &testServer{}
	go func() {
		err := http.ListenAndServe(":9092", s0)
		if err != nil {
			t.Fatal(err)
		}
	}()
	//
	sv := Default(&Config{
		Debug: true,
	})
	r0 := route.Route{
		Domain: "example.com",
		Server: route.RouteServer{
			OutConnType: route.HTTP,
			OutAddress:  "localhost:9092",
		},
	}
	err := sv.Add(r0)
	if err != nil {
		t.Fatal(err)
	}
	r0.Domain = "example.net"
	err = sv.Add(r0)
	if err != nil {
		t.Fatal(err)
	}

	w, r := testRequest("example.com", "GET", "/")

	sv.ServeHTTP(w, r)
	if w.Body.String() != "Hello!" {
		t.Fatalf("(example.com) Body should be %v but it is %v", "Hello!", w.Body.String())
	}

	w, r = testRequest("example.net", "GET", "/")

	sv.ServeHTTP(w, r)
	if w.Body.String() != "Hello!" {
		t.Fatalf("(example.net) Body should be %v but it is %v", "Hello!", w.Body.String())
	}

	w, r = testRequest("notfound.net", "GET", "/")

	sv.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("(notfound.net) Status should be %v but it is %v", http.StatusInternalServerError, w.Code)
	}
}

func TestServerSSL(t *testing.T) {
	//
	// start dummy server
	s0 := &testServer{}
	go func() {
		err := http.ListenAndServe(":9095", s0)
		if err != nil {
			t.Fatal(err)
		}
	}()
	//
	sv := Default(&Config{
		Debug:         true,
		ListenAddrTLS: ":9093",
		ListenAddr:    ":9098",
	})
	r0 := route.Route{
		Domain: "example.com",
		Server: route.RouteServer{
			OutConnType: route.HTTP,
			OutAddress:  "localhost:9095",
		},
		Certificate: util.Certificate{
			CertFile: "../testfiles/example.com.cert.pem",
			KeyFile:  "../testfiles/example.com.key.pem",
		},
	}
	err := sv.Add(r0)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err := sv.Run()
		if err != nil {
			t.Fatal(err)
		}
	}()

	time.Sleep(time.Millisecond * 250)

	_, r := testRequest("example.com", "GET", "https://localhost:9093/")
	cl := http.Client{}
	cl.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	resp, err := cl.Do(r)
	if err != nil {
		t.Log(r.URL)
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		bf := make([]byte, 512)
		resp.Body.Read(bf)
		t.Fatalf("Status should be 200 but it is %v '%s'", resp.StatusCode, string(bf))
	}

}

func TestRedirect(t *testing.T) {
	sv := Default(&Config{
		FallbackDomain: "a",
		ListenAddr:     ":9887",
		DisableTLS:     true,
	})
	r0 := route.Route{
		Domain: "a",
		Server: route.RouteServer{
			OutConnType: route.REDIRECT,
			OutAddress:  "https://www.google.com",
		},
	}
	sv.Add(r0)
	go func() {
		err := sv.Run()
		if err != nil {
			t.Fatal(err)
		}
	}()
	time.Sleep(time.Millisecond * 250)

	_, r := testRequest("example.com", "GET", "http://localhost:9887/")
	cl := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := cl.Do(r)
	if err != nil {
		t.Fatal(err)
		return
	}
	if resp.StatusCode != http.StatusPermanentRedirect {
		t.Fatal("http status code not 301", resp.Status, resp.StatusCode, resp.Header)
	}
}
