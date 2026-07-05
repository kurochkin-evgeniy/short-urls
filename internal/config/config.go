// Package config загружает настройки приложения из флагов командной строки и переменных окружения.
package config

import (
	"flag"
	"os"
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

// NewConfig разбирает флаги и переменные окружения и возвращает Config.
func NewConfig() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.HostAddr, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.BaseUrl, "b", "http://localhost:8080", "Base URL for shortened links")
	flag.StringVar(&cfg.FilePath, "f", "", "storage path")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "PostgreSQL DSN")
	flag.BoolVar(&cfg.EnableHTTPS, "s", false, "enable HTTPS")
	flag.StringVar(&cfg.TLSCertFile, "c", "", "TLS certificate file path")
	flag.StringVar(&cfg.TLSKeyFile, "k", "", "TLS key file path")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "audit log file path")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "audit remote server URL")
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

	return cfg
}
