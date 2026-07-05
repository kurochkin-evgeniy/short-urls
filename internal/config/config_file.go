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
	if fileCfg.ServerAddress != nil {
		cfg.HostAddr = *fileCfg.ServerAddress
	}
	if fileCfg.BaseURL != nil {
		cfg.BaseUrl = *fileCfg.BaseURL
	}
	if fileCfg.FileStoragePath != nil {
		cfg.FilePath = *fileCfg.FileStoragePath
	}
	if fileCfg.DatabaseDSN != nil {
		cfg.DatabaseDSN = *fileCfg.DatabaseDSN
	}
	if fileCfg.CookieSecret != nil {
		cfg.CookieSecret = *fileCfg.CookieSecret
	}
	if fileCfg.AuditFile != nil {
		cfg.AuditFile = *fileCfg.AuditFile
	}
	if fileCfg.AuditURL != nil {
		cfg.AuditURL = *fileCfg.AuditURL
	}
	if fileCfg.EnableHTTPS != nil {
		cfg.EnableHTTPS = *fileCfg.EnableHTTPS
	}
	if fileCfg.TLSCertFile != nil {
		cfg.TLSCertFile = *fileCfg.TLSCertFile
	}
	if fileCfg.TLSKeyFile != nil {
		cfg.TLSKeyFile = *fileCfg.TLSKeyFile
	}
}
