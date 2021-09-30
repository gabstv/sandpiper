package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/gabstv/sandpiper/internal/pkg/envs"
	"github.com/gabstv/sandpiper/internal/pkg/route"
	"github.com/gabstv/sandpiper/pkg/server"
	"github.com/gabstv/sandpiper/pkg/util"
	colorable "github.com/mattn/go-colorable"
	"github.com/mgutz/ansi"
	yaml "gopkg.in/yaml.v2"
)

var (
	configfile string
	stdout     io.Writer
	stderr     io.Writer
)

func main() {
	stdout = colorable.NewColorableStdout()
	stderr = colorable.NewColorableStderr()
	fmt.Fprintln(stdout, ansi.Color("\nSANDPIPER 1.4.3\n", "green"))
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

	cfg := &Config{}
	err = yaml.Unmarshal(bs, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: Unable to unmarshal the config file at %v!\n%v\n",
			ansi.Color(configfile, "red"),
			ansi.Color(err.Error(), "red"))
		os.Exit(1)
	}

	// unpack Config
	svCfg := &server.Config{}
	svCfg.Debug = cfg.Debug
	if svCfg.Debug {
		util.DEBUG = true
	}
	svCfg.ListenAddr = cfg.ListenAddr
	svCfg.ListenAddrTLS = cfg.ListenAddrTLS
	svCfg.DisableTLS = cfg.DisableTLS
	svCfg.NumCPU = cfg.NumCPU
	svCfg.FallbackDomain = cfg.FallbackDomain
	svCfg.Graceful = cfg.Graceful
	svCfg.CachePath = cfg.CachePath
	svCfg.APIListen = cfg.APIListen
	svCfg.APIKey = cfg.APIKey
	svCfg.APIDomain = cfg.APIDomain
	svCfg.APIDomainAutocert = cfg.APIDomainAutocert
	svCfg.APIIndexFile = cfg.APIIndexFile
	svCfg.APIHostFolders = cfg.APIHostFolders
	//
	svCfg.S3Cache = cfg.S3Cache
	svCfg.S3ID = cfg.S3ID
	svCfg.S3Secret = cfg.S3Secret
	svCfg.S3Region = cfg.S3Region
	svCfg.S3Bucket = cfg.S3Bucket
	svCfg.S3Folder = cfg.S3Folder
	svCfg.AutocertAll = cfg.AutocertAll

	// ENV VARS
	if dbg, ok := envs.Debug(); ok {
		svCfg.Debug = dbg
	}
	if vv := envs.Listen(); vv != "" {
		svCfg.ListenAddr = vv
	}
	if vv := envs.ListenTLS(); vv != "" {
		svCfg.ListenAddrTLS = vv
	}
	if vv := envs.FallbackDomain(); vv != "" {
		svCfg.FallbackDomain = vv
	}

	if vv := os.Getenv("API_LISTEN"); vv != "" {
		svCfg.APIListen = vv
	}
	if vv := os.Getenv("API_KEY"); vv != "" {
		svCfg.APIKey = vv
	}
	if vv := os.Getenv("API_DOMAIN"); vv != "" {
		svCfg.APIDomain = vv
	}
	if vv := os.Getenv("API_DOMAIN_AUTOCERT"); vv != "" {
		svCfg.APIDomainAutocert = (vv == "1")
	}
	if vv := os.Getenv("API_INDEX_FILE"); vv != "" {
		svCfg.APIIndexFile = vv
	}
	if vv := os.Getenv("API_HOST_FOLDERS"); vv != "" {
		svCfg.APIHostFolders = strings.Split(vv, ",")
	}
	if vv := os.Getenv("LETSENCRYPT_URL"); vv != "" {
		svCfg.LetsEncryptURL = vv
	}
	if vv := os.Getenv("DISABLE_TLS"); vv != "" {
		svCfg.DisableTLS = (vv == "1")
	}
	if vv := os.Getenv("S3_CACHE"); vv != "" {
		svCfg.S3Cache = (vv == "1")
	}
	if vv := os.Getenv("S3_ID"); vv != "" {
		svCfg.S3ID = vv
	}
	if vv := os.Getenv("S3_SECRET"); vv != "" {
		svCfg.S3Secret = vv
	}
	if vv := os.Getenv("S3_REGION"); vv != "" {
		svCfg.S3Region = vv
	}
	if vv := os.Getenv("S3_BUCKET"); vv != "" {
		svCfg.S3Bucket = vv
	}
	if vv := os.Getenv("S3_FOLDER"); vv != "" {
		svCfg.S3Folder = vv
	}
	if vv := os.Getenv("AUTOCERT_ALL"); vv != "" {
		svCfg.AutocertAll = (vv == "1")
	}
	if vv := os.Getenv("FALLBACK_DOMAIN"); vv != "" {
		svCfg.FallbackDomain = vv
	}

	s := server.Default(svCfg)

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
		r.AuthMode = v.AuthMode
		r.AuthKey = v.AuthKey
		r.AuthValue = v.AuthValue
		r.ForceHTTPS = v.ForceHTTPS
		r.Server.LoadBalancer = v.LoadBalancer
		r.FlushInterval = v.FlushInterval
		if v.Domain == "" && v.Domains != nil && len(v.Domains) > 0 {
			for _, dname := range v.Domains {
				r2 := r
				r2.Domain = dname
				err = s.Add(r2)
				if err != nil {
					fmt.Fprintf(stderr, "\nERROR: Could not add route %v\n%v\n",
						ansi.Color(dname, "yellow"),
						ansi.Color(err.Error(), "red"))
					os.Exit(1)
				}
			}
			fmt.Fprintf(stdout, "%v: %v\n",
				ansi.Color("Domain added", "green"),
				strings.Join(v.Domains, " "))
		} else {
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
	}
	// ROUTES BY ENV VARS
	if evroutes := os.Getenv("ENV_ROUTES"); evroutes != "" {
		evn, _ := strconv.Atoi(evroutes)
		for i := 0; i < evn; i++ {
			if v := os.Getenv(fmt.Sprintf("R%d_DOMAIN", i)); v == "" {
				fmt.Fprintf(stderr, "\nERROR: Could not add env route %v\n%v\n",
					ansi.Color(strconv.Itoa(i), "yellow"),
					ansi.Color(fmt.Sprintf("invalid R%d_DOMAIN var", i), "red"))
				continue
			}
			r := route.Route{}
			if v := os.Getenv(fmt.Sprintf("R%d_OUT_CONN_TYPE", i)); v != "" {
				r.Server.OutConnType = route.ParseConnType(v)
			}
			if v := os.Getenv(fmt.Sprintf("R%d_OUT_ADDR", i)); v != "" {
				r.Server.OutAddress = v
			}
			if v := os.Getenv(fmt.Sprintf("R%d_TLS_CERT_FILE", i)); v != "" {
				r.Certificate.CertFile = v
			}
			if v := os.Getenv(fmt.Sprintf("R%d_TLS_KEY_FILE", i)); v != "" {
				r.Certificate.KeyFile = v
			}
			if v := os.Getenv(fmt.Sprintf("R%d_AUTOCERT", i)); v == "1" || v == "true" || v == "TRUE" {
				r.Autocert = true
			}
			if v := os.Getenv(fmt.Sprintf("R%d_AUTH_MODE", i)); v != "" {
				r.AuthMode = v
			}
			if v := os.Getenv(fmt.Sprintf("R%d_AUTH_KEY", i)); v != "" {
				r.AuthKey = v
			}
			if v := os.Getenv(fmt.Sprintf("R%d_AUTH_VALUE", i)); v != "" {
				r.AuthValue = v
			}
			if v := os.Getenv(fmt.Sprintf("R%d_FORCE_HTTPS", i)); v == "1" || v == "true" || v == "TRUE" {
				r.ForceHTTPS = true
			}
			if v := os.Getenv(fmt.Sprintf("R%d_FLUSH_INTERVAL", i)); v != "" {
				r.FlushInterval, _ = strconv.Atoi(v)
			}
			r.WsCFG.Enabled = true
			r.WsCFG.ReadBufferSize = 1024 * 8
			r.WsCFG.WriteBufferSize = 1024 * 8

			if v := os.Getenv(fmt.Sprintf("R%d_DOMAIN", i)); v != "" {
				dms := strings.Split(v, ";")
				for _, kv := range dms {
					r2 := r
					r2.Domain = strings.TrimSpace(kv)
					err = s.Add(r2)
					if err != nil {
						fmt.Fprintf(stderr, "\nERROR: Could not add route %v\n%v\n",
							ansi.Color(kv, "yellow"),
							ansi.Color(err.Error(), "red"))
						os.Exit(1)
					}
				}
			}
		}
	}
	//
	if cfg.Debug {
		fmt.Fprintf(stdout, "%v DEBUG MODE IS %v\n",
			ansi.Color("WARNING:", "yellow"),
			ansi.Color("ON", "green"))
	}
	//
	// Close if received signal
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, os.Interrupt, os.Kill)
		<-sigchan
		s.Close()
	}()
	//
	err = s.Run()
	if err != nil {
		fmt.Fprintf(stderr, "ERROR (s.Run()): %v\n",
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
	if input == "REDIRECT" {
		return route.REDIRECT, true
	}
	if input == "LOAD_BALANCER" {
		return route.LOAD_BALANCER, true
	}
	return route.HTTP, false
}

// Config main config structure (yml)
type Config struct {
	Debug             bool          `yaml:"debug"`
	NumCPU            int           `yaml:"num_cpu"`
	ListenAddr        string        `yaml:"listen_addr"`
	ListenAddrTLS     string        `yaml:"listen_addr_tls"`
	DisableTLS        bool          `yaml:"disable_tls"`
	Routes            []ConfigRoute `yaml:"routes"`
	FallbackDomain    string        `yaml:"fallback_domain"`
	Graceful          bool          `yaml:"graceful"`
	CachePath         string        `yaml:"cache_path"`
	APIListen         string        `yaml:"api_listen"`
	APIKey            string        `yaml:"api_key"`
	APIDomain         string        `yaml:"api_domain"`
	APIDomainAutocert bool          `yaml:"api_domain_autocert"`
	APIIndexFile      string        `yaml:"api_index_file"`
	APIHostFolders    []string      `yaml:"api_host_folders"`
	S3Cache           bool          `yaml:"s3_cache"`
	S3ID              string        `yaml:"s3_id"`
	S3Secret          string        `yaml:"s3_secret"`
	S3Region          string        `yaml:"s3_region"`
	S3Bucket          string        `yaml:"s3_bucket"`
	S3Folder          string        `yaml:"s3_folder"`
	AutocertAll       bool          `yaml:"autocert_all"`
}

// ConfigRoute represents a domain route
type ConfigRoute struct {
	Domain                 string                    `yaml:"domain"`
	Domains                []string                  `yaml:"domains"`
	OutgoingServerConnType string                    `yaml:"out_conn_type"`
	OutgoingServerAddress  string                    `yaml:"out_addr"`
	TLSCertFile            string                    `yaml:"tls_cert_file"`
	TLSKeyFile             string                    `yaml:"tls_key_file"`
	Autocert               bool                      `yaml:"autocert"`
	Websockets             util.WsConfig             `yaml:"websockets"`
	AuthMode               string                    `yaml:"auth_mode"`
	AuthKey                string                    `yaml:"auth_key"`
	AuthValue              string                    `yaml:"auth_value"`
	ForceHTTPS             bool                      `yaml:"force_https"`
	LoadBalancer           *route.LoadBalancerConfig `yaml:"load_balancer"`
	FlushInterval          int                       `yaml:"flush_interval"`
}
