package config

import "flag"

type Config struct {
	HostAddr string
	BaseUrl  string
}

func NewConfig() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.HostAddr, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.BaseUrl, "b", "http://localhost:8080", "Base URL for shortened links")
	flag.Parse()
	return cfg
}
