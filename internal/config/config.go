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

	if hostAddr != "" {
		cfg.HostAddr = hostAddr
	}
	if baseURL != "" {
		cfg.BaseUrl = baseURL
	}
	if filePath != "" {
		cfg.FilePath = filePath
	}
	if databaseDSN != "" {
		cfg.DatabaseDSN = databaseDSN
	}
	if tlsKeyFile != "" {
		cfg.TLSKeyFile = tlsKeyFile
	}
	if auditFile != "" {
		cfg.AuditFile = auditFile
	}
	if auditURL != "" {
		cfg.AuditURL = auditURL
	}
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "s" {
			cfg.EnableHTTPS = enableHTTPS
		}
	})

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
