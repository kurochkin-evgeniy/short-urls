// Package config загружает настройки приложения из файла, флагов командной строки и переменных окружения.
package config

import (
	"flag"
	"os"
	"strconv"
)

// Config содержит параметры запуска сервиса сокращения URL.
type Config struct {
	HostAddr     string
	BaseUrl      string
	FilePath     string
	DatabaseDSN  string
	CookieSecret string
	AuditFile    string
	AuditURL     string
	EnableHTTPS  bool
	TLSCertFile  string
	TLSKeyFile   string
}

func defaultConfig() *Config {
	return &Config{
		HostAddr: "localhost:8080",
		BaseUrl:  "http://localhost:8080",
	}
}

// NewConfig разбирает файл конфигурации, флаги и переменные окружения и возвращает Config.
// Приоритет: значения по умолчанию < файл < переменные окружения < флаги.
func NewConfig() (*Config, error) {
	cfg := defaultConfig()

	var configFile string
	flag.StringVar(&configFile, "c", "", "config file path")
	flag.StringVar(&configFile, "config", "", "config file path")
	flag.StringVar(&cfg.HostAddr, "a", cfg.HostAddr, "HTTP server address")
	flag.StringVar(&cfg.BaseUrl, "b", cfg.BaseUrl, "Base URL for shortened links")
	flag.StringVar(&cfg.FilePath, "f", "", "storage path")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "PostgreSQL DSN")
	flag.BoolVar(&cfg.EnableHTTPS, "s", false, "enable HTTPS")
	flag.StringVar(&cfg.TLSKeyFile, "k", "", "TLS key file path")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "audit log file path")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "audit remote server URL")
	flag.Parse()

	visited := map[string]bool{}
	flag.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})

	configPath := ""
	if visited["c"] || visited["config"] {
		configPath = configFile
	} else if val, ok := os.LookupEnv("CONFIG"); ok {
		configPath = val
	}

	cfg = defaultConfig()
	if configPath != "" {
		if err := loadConfigFromFile(configPath, cfg); err != nil {
			return nil, err
		}
	}

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "c", "config":
			return
		case "a":
			cfg.HostAddr = f.Value.String()
		case "b":
			cfg.BaseUrl = f.Value.String()
		case "f":
			cfg.FilePath = f.Value.String()
		case "d":
			cfg.DatabaseDSN = f.Value.String()
		case "s":
			enabled, err := strconv.ParseBool(f.Value.String())
			if err != nil {
				return
			}
			cfg.EnableHTTPS = enabled
		case "k":
			cfg.TLSKeyFile = f.Value.String()
		case "audit-file":
			cfg.AuditFile = f.Value.String()
		case "audit-url":
			cfg.AuditURL = f.Value.String()
		}
	})

	applyEnv(cfg)

	return cfg, nil
}

func applyEnv(cfg *Config) {
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
	if val, ok := os.LookupEnv("COOKIE_SECRET"); ok {
		cfg.CookieSecret = val
	}
	if val, ok := os.LookupEnv("AUDIT_FILE"); ok {
		cfg.AuditFile = val
	}
	if val, ok := os.LookupEnv("AUDIT_URL"); ok {
		cfg.AuditURL = val
	}
	if _, ok := os.LookupEnv("ENABLE_HTTPS"); ok {
		cfg.EnableHTTPS = true
	}
	if val, ok := os.LookupEnv("TLS_CERT_FILE"); ok {
		cfg.TLSCertFile = val
	}
	if val, ok := os.LookupEnv("TLS_KEY_FILE"); ok {
		cfg.TLSKeyFile = val
	}
}
