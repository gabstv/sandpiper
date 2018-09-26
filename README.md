Sandpiper: A fast reverse proxy server
======================================

Sandpiper is an open source websocket-aware reverse proxy server.

## Installation

To install from source (you need to have Go 1.4+ installed):

```go
go get github.com/gabstv/sandpiper/cmd/sandpiper
```

## Setup

After you installed Sandpiper, use the following command to run:

```bash
sandpiper /path/to/config.yml
```

There are other ways to retrieve the configuration file. Run `sandpiper help` to learn more.

## Configure

The configuration file is in the [YAML](http://yaml.org/) format.

Example:

```yaml
debug: true
# If num_cpu is set to 0, or absent, it will default to the
# amount of CPUs available.
num_cpu: 0
listen_addr:     :8080
listen_addr_tls: :8443
graceful:        true           # graceful shutdown (on interrupt)
cache_path:      /tmp/sandpiper # a place to store autocerts
routes:
  - 
    domain:        example.com
    out_conn_type: HTTP
    out_addr:      localhost:8011
    websockets:
      enabled:               true
      read_buffer_size:      2048
      write_buffer_size:     2048
      read_deadline_seconds: 60
  - 
    domain:        example.net
    out_conn_type: HTTP
    out_addr:      localhost:8012
    tls_cert_file: example.net.cert.pem
    tls_key_file:  example.net.key.pem
  - 
    domain:        auto.ssl.cert.by.lestencrypt.example.org
    out_conn_type: HTTP
    out_addr:      localhost:8013
    autocert:      true
    # use autocert to generate a domain validated certificate automatically via LetsEncrypt
```