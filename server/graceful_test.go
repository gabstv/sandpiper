package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gabstv/sandpiper/route"

	"github.com/gabstv/freeport"
)

type delayTestServer struct {
}

func (t *delayTestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("delayTestServer Hello!\n"))
	time.Sleep(time.Second)
	w.Write([]byte("delayTestServer 1\n"))
	time.Sleep(time.Second)
	w.Write([]byte("delayTestServer 2\n"))
	time.Sleep(time.Second)
	w.Write([]byte("delayTestServer 3\n"))
	time.Sleep(time.Second)
	w.Write([]byte("delayTestServer 4\n"))
	time.Sleep(time.Second)
	w.Write([]byte("delayTestServer 5\n"))
}

func TestShutdown(t *testing.T) {

	site1Port, err := freeport.TCP()
	if err != nil {
		t.Error(err)
	}

	mainPort, err := freeport.TCP()
	if err != nil {
		t.Error(err)
	}

	if site1Port == mainPort {
		t.Errorf("ports should not be equal\n")
	}

	sv := Default(&Config{
		Graceful:       true,
		Debug:          true,
		ListenAddr:     fmt.Sprintf("localhost:%v", mainPort),
		FallbackDomain: fmt.Sprintf("localhost:%v", site1Port),
	})

	rsite1 := route.Route{
		Domain: "site1.com",
		Server: route.RouteServer{
			OutConnType: route.HTTP,
			OutAddress:  fmt.Sprintf("localhost:%v", site1Port),
		},
	}

	var wg sync.WaitGroup

	site1 := &delayTestServer{}
	go func() {
		//wg.Add(1)
		//defer wg.Done()
		err := http.ListenAndServe(rsite1.Server.OutAddress, site1)
		if err != nil {
			t.Fatal(err)
		}
	}()

	sv.Add(rsite1)
	go sv.Run()
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%v/", mainPort), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Sandpiper-Host", rsite1.Domain)
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Second * 1)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Client connect error: %v\n", err.Error())
		} else {
			if resp.StatusCode != 200 {
				t.Fatal(resp.Status)
			}
			io.Copy(os.Stdout, resp.Body)
			resp.Body.Close()
		}
	}()
	go func() {
		time.Sleep(time.Second * 3)
		sv.Close()
	}()

	wg.Wait()
	fmt.Println("SHUTDOWN")
}
