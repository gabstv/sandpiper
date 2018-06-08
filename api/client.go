package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gabstv/sandpiper/route"
)

// Client is the API client.
type Client struct {
	APIKey   string
	Endpoint string
}

// NewClient creates a api client with the specified key and endpoint
func NewClient(apikey, endpoint string) *Client {
	return &Client{
		APIKey:   apikey,
		Endpoint: endpoint,
	}
}

// PutRoute adds a new route and returns the local tcp port to use.
// The port is only returned if the out_type is set to "auto".
func (c *Client) PutRoute(route NewRoute) (port int, err error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(&route); err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodPut, c.Endpoint+"/v1/route", buf)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-API-KEY", c.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	jd := struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
		Port    int    `json:"port,omitempty"`
	}{}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&jd); err != nil {
		return 0, err
	}
	if !jd.Success {
		return 0, fmt.Errorf(jd.Error)
	}
	return jd.Port, nil
}

// GetRoutes returns all the registered routes.
func (c *Client) GetRoutes() (map[string]route.Route, error) {
	req, err := http.NewRequest(http.MethodGet, c.Endpoint+"/v1/routes", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", c.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	jd := make(map[string]route.Route)
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&jd); err != nil {
		return nil, err
	}
	return jd, nil
}
