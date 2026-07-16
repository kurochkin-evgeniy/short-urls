// Package config загружает настройки приложения из файла, флагов командной строки и переменных окружения.
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

	var (
		configFile  string
		hostAddr    string
		baseURL     string
		filePath    string
		databaseDSN string
		enableHTTPS bool
		tlsKeyFile  string
		auditFile   string
		auditURL    string
	)

	flag.StringVar(&configFile, "c", "", "config file path")
	flag.StringVar(&configFile, "config", "", "config file path")
	flag.StringVar(&hostAddr, "a", "", "HTTP server address")
	flag.StringVar(&baseURL, "b", "", "Base URL for shortened links")
	flag.StringVar(&filePath, "f", "", "storage path")
	flag.StringVar(&databaseDSN, "d", "", "PostgreSQL DSN")
	flag.BoolVar(&enableHTTPS, "s", false, "enable HTTPS")
	flag.StringVar(&tlsKeyFile, "k", "", "TLS key file path")
	flag.StringVar(&auditFile, "audit-file", "", "audit log file path")
	flag.StringVar(&auditURL, "audit-url", "", "audit remote server URL")
	flag.Parse()

	configPath := configFile
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}
	if configPath != "" {
		if err := loadConfigFromFile(configPath, cfg); err != nil {
			return nil, err
		}
	}

	applyEnv(cfg)

	setIfNotEmpty(&cfg.HostAddr, hostAddr)
	setIfNotEmpty(&cfg.BaseUrl, baseURL)
	setIfNotEmpty(&cfg.FilePath, filePath)
	setIfNotEmpty(&cfg.DatabaseDSN, databaseDSN)
	setIfNotEmpty(&cfg.TLSKeyFile, tlsKeyFile)
	setIfNotEmpty(&cfg.AuditFile, auditFile)
	setIfNotEmpty(&cfg.AuditURL, auditURL)
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "s" {
			cfg.EnableHTTPS = enableHTTPS
		}
	})

	return cfg, nil
}

func applyEnv(cfg *Config) {
	setFromEnv(&cfg.HostAddr, "SERVER_ADDRESS")
	setFromEnv(&cfg.BaseUrl, "BASE_URL")
	setFromEnv(&cfg.FilePath, "FILE_STORAGE_PATH")
	setFromEnv(&cfg.DatabaseDSN, "DATABASE_DSN")
	setFromEnv(&cfg.CookieSecret, "COOKIE_SECRET")
	setFromEnv(&cfg.AuditFile, "AUDIT_FILE")
	setFromEnv(&cfg.AuditURL, "AUDIT_URL")
	if _, ok := os.LookupEnv("ENABLE_HTTPS"); ok {
		cfg.EnableHTTPS = true
	}
	setFromEnv(&cfg.TLSCertFile, "TLS_CERT_FILE")
	setFromEnv(&cfg.TLSKeyFile, "TLS_KEY_FILE")
}

func setFromPtr[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
}

func setIfNotEmpty(dst *string, src string) {
	if src != "" {
		*dst = src
	}
}

func setFromEnv(dst *string, key string) {
	if val, ok := os.LookupEnv(key); ok {
		*dst = val
	}
}
