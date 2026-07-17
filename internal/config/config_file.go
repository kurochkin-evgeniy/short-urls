package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type fileConfig struct {
	ServerAddress   *string `json:"server_address"`
	BaseURL         *string `json:"base_url"`
	FileStoragePath *string `json:"file_storage_path"`
	DatabaseDSN     *string `json:"database_dsn"`
	CookieSecret    *string `json:"cookie_secret"`
	AuditFile       *string `json:"audit_file"`
	AuditURL        *string `json:"audit_url"`
	EnableHTTPS     *bool   `json:"enable_https"`
	TLSCertFile     *string `json:"tls_cert_file"`
	TLSKeyFile      *string `json:"tls_key_file"`
}

func loadConfigFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var fileCfg fileConfig
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	applyFileConfig(&fileCfg, cfg)
	return nil
}

func applyFileConfig(fileCfg *fileConfig, cfg *Config) {
	setFromPtr(&cfg.HostAddr, fileCfg.ServerAddress)
	setFromPtr(&cfg.BaseUrl, fileCfg.BaseURL)
	setFromPtr(&cfg.FilePath, fileCfg.FileStoragePath)
	setFromPtr(&cfg.DatabaseDSN, fileCfg.DatabaseDSN)
	setFromPtr(&cfg.CookieSecret, fileCfg.CookieSecret)
	setFromPtr(&cfg.AuditFile, fileCfg.AuditFile)
	setFromPtr(&cfg.AuditURL, fileCfg.AuditURL)
	setFromPtr(&cfg.EnableHTTPS, fileCfg.EnableHTTPS)
	setFromPtr(&cfg.TLSCertFile, fileCfg.TLSCertFile)
	setFromPtr(&cfg.TLSKeyFile, fileCfg.TLSKeyFile)
}
