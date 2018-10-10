package envs

import (
	"os"
)

// Debug -> DEBUG
func Debug() (debug, ok bool) {
	dbg := os.Getenv("DEBUG")
	if dbg == "" {
		return false, false
	}
	return dbg == "1", true
}

// Listen -> LISTEN
func Listen() string {
	return os.Getenv("LISTEN")
}

// ListenTLS -> LISTEN_TLS
func ListenTLS() string {
	return os.Getenv("LISTEN_TLS")
}

// FallbackDomain -> FALLBACK_DOMAIN
func FallbackDomain() string {
	return os.Getenv("FALLBACK_DOMAIN")
}
