package main

import (
	"flag"

	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/api"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/config"
	"github.com/thomas-maurice/goapp-mutating-webhook/pkg/log"
)

var (
	flagAddr   string
	flagKey    string
	flagCert   string
	flagConfig string
)

func init() {
	flag.StringVar(&flagAddr, "addr", ":443", "Address to bind to")
	flag.StringVar(&flagCert, "cert", "cert.pem", "Certificate to use")
	flag.StringVar(&flagKey, "key", "key.pem", "Key to use")
	flag.StringVar(&flagConfig, "config", "config.yaml", "Config file to use")
}

func main() {
	flag.Parse()

	cfg, err := config.GetConfigFromFile(flagConfig)
	if err != nil {
		panic(err)
	}

	api, err := api.NewAPI(log.GetLogger(), cfg)
	if err != nil {
		panic(err)
	}

	err = api.Serve(flagAddr, flagCert, flagKey)
	if err != nil {
		panic(err)
	}
}
