package server

// Config contains the root configuration of a sandpiper server
type Config struct {
	Debug bool
	// Default: max cpu (0)
	NumCPU         int
	ListenAddr     string
	ListenAddrTLS  string
	DisableTLS     bool
	FallbackDomain string
	Graceful       bool
	CachePath      string
	// Change this to use a different letsencrypt url for handling cert requests.
	LetsEncryptURL string
	// APIListen lets you host the REST api on "host:port"
	// The API will not be available if the config is empty.
	APIListen string
	// API key to use for API commands/communication
	APIKey string
	// If not empty, APIDomain will spin a domain/route for the API
	APIDomain         string
	APIDomainAutocert bool
}
