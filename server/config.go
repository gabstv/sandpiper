package server

type Config struct {
	Debug          bool
	NumCPU         int
	ListenAddr     string
	ListenAddrTLS  string
	FallbackDomain string
	Graceful       bool
	CachePath      string
}
