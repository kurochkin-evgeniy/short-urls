package config

import (
	"flag"
	"os"
)

type Config struct {
	HostAddr    string
	BaseUrl     string
	FilePath    string
	DatabaseDSN string
}

func NewConfig() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.HostAddr, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.BaseUrl, "b", "http://localhost:8080", "Base URL for shortened links")
	flag.StringVar(&cfg.FilePath, "f", "", "storage path")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "PostgreSQL DSN")
	flag.Parse()

	if val, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		cfg.HostAddr = val
	}

	if val, ok := os.LookupEnv("BASE_URL"); ok {
		cfg.BaseUrl = val
	}

	if val, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FilePath = val
	}

	if val, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = val
	}

	return cfg
}
