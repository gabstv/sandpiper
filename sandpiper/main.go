package main

import (
	"flag"
	"fmt"
	"github.com/gabstv/sandpiper/route"
	"github.com/gabstv/sandpiper/server"
	"github.com/gabstv/sandpiper/util"
	"github.com/mattn/go-colorable"
	"github.com/mgutz/ansi"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
)

var (
	configfile string
	stdout     io.Writer
	stderr     io.Writer
)

func main() {
	stdout = colorable.NewColorableStdout()
	stderr = colorable.NewColorableStderr()
	fmt.Fprintln(stdout, ansi.Color("\nSANDPIPER 1.1.0\n", "green"))
	// init flags
	flag.Parse()
	// print help if needed
	printHelp()
	// find config
	findConfigFile()

	bs, err := ioutil.ReadFile(configfile)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: Unable to load the config file at %v!\n%v\n",
			ansi.Color(configfile, "red"),
			ansi.Color(err.Error(), "red"))
		os.Exit(1)
	}
	s := server.Default()

	cfg := &Config{}
	err = yaml.Unmarshal(bs, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: Unable to unmarshal the config file at %v!\n%v\n",
			ansi.Color(configfile, "red"),
			ansi.Color(err.Error(), "red"))
		os.Exit(1)
	}
	// unpack Config
	s.Cfg.Debug = cfg.Debug
	if s.Cfg.Debug {
		util.DEBUG = true
	}
	s.Cfg.ListenAddr = cfg.ListenAddr
	s.Cfg.ListenAddrTLS = cfg.ListenAddrTLS
	s.Cfg.NumCPU = cfg.NumCPU
	s.Cfg.FallbackDomain = cfg.FallbackDomain
	s.Cfg.Graceful = cfg.Graceful
	for _, v := range cfg.Routes {
		r := route.Route{}
		// apply Websockets config
		r.WsCFG = v.Websockets
		if ct, ok := unpackConnType(v.OutgoingServerConnType); ok {
			r.Server.OutConnType = ct
		} else {
			fmt.Fprintf(stderr, "\n CONFIGURATION ERROR\nDomain: %v\n ERR: %v\n",
				ansi.Color(v.Domain, "yellow"),
				ansi.Color("Invalid conn type "+v.OutgoingServerConnType, "red"))
			os.Exit(1)
		}
		r.Domain = v.Domain
		r.Server.OutAddress = v.OutgoingServerAddress
		r.Certificate.CertFile = v.TLSCertFile
		r.Certificate.KeyFile = v.TLSKeyFile
		r.Autocert = v.Autocert
		err = s.Add(r)
		if err != nil {
			fmt.Fprintf(stderr, "\nERROR: Could not add route %v\n%v\n",
				ansi.Color(v.Domain, "yellow"),
				ansi.Color(err.Error(), "red"))
			os.Exit(1)
		}
		fmt.Fprintf(stdout, "%v: %v\n",
			ansi.Color("Domain added", "green"),
			v.Domain)
	}
	//
	if cfg.Debug {
		fmt.Fprintf(stdout, "%v DEBUG MODE IS %v\n",
			ansi.Color("WARNING:", "yellow"),
			ansi.Color("ON", "green"))
	}
	//
	err = s.Run()
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n",
			ansi.Color(err.Error(), "red"))
		os.Exit(1)
	}
}

func printHelp() {
	if len(flag.Args()) < 1 {
		return
	}
	if flag.Arg(0) != "help" {
		return
	}
	fmt.Fprintln(stdout, ansi.Color("Usage:", "blue"))
	fmt.Fprintf(stdout, "sandpiper %v\n", ansi.Color("config.yml", "green"))
	fmt.Fprintf(stdout, "sandpiper %v\n", ansi.Color("help", "green"))
	os.Exit(0)
}

func findConfigFile() {
	if len(flag.Args()) > 0 {
		configfile = flag.Arg(0)
		return
	}
	if env := os.Getenv("SANDPIPER_CONFIG"); env != "" {
		configfile = env
		return
	}
	if _, err := os.Stat("config.yml"); err == nil {
		configfile = "config.yml"
		return
	}
	// no config file found!
	fmt.Fprint(stderr, ansi.Color("No configuration file found!\n\n", "red"))
	fmt.Println(`There are three different ways to fix this:
  - Pass it as a parameter:
     sandpiper /path/to/config.yml
  - Set the env var SANDPIPER_CONFIG
  - Have a config.yml in the current working directory`)
	fmt.Fprintf(stdout, "\nrun %v fo more info.\n", ansi.Color("sandpiper help", "blue"))
	os.Exit(1)
}

func unpackConnType(input string) (route.ConnType, bool) {
	if input == "HTTP" {
		return route.HTTP, true
	}
	if input == "HTTPS" || input == "HTTPS_VERIFY" {
		return route.HTTPS_VERIFY, true
	}
	if input == "HTTPS_SKIP_VERIFY" {
		return route.HTTPS_SKIP_VERIFY, true
	}
	return route.HTTP, false
}

type Config struct {
	Debug          bool          `yaml:"debug"`
	NumCPU         int           `yaml:"num_cpu"`
	ListenAddr     string        `yaml:"listen_addr"`
	ListenAddrTLS  string        `yaml:"listen_addr_tls"`
	Routes         []ConfigRoute `yaml:"routes"`
	FallbackDomain string        `yaml:"fallback_domain"`
	Graceful       bool          `yaml:"graceful"`
}

type ConfigRoute struct {
	Domain                 string        `yaml:"domain"`
	OutgoingServerConnType string        `yaml:"out_conn_type"`
	OutgoingServerAddress  string        `yaml:"out_addr"`
	TLSCertFile            string        `yaml:"tls_cert_file"`
	TLSKeyFile             string        `yaml:"tls_key_file"`
	Autocert               bool          `yaml:"autocert"`
	Websockets             util.WsConfig `yaml:"websockets"`
}
