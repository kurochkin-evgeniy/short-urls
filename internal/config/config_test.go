package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetFlags(t *testing.T) {
	t.Helper()
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestNewConfigDefaults(t *testing.T) {
	resetFlags(t)
	os.Unsetenv("SERVER_ADDRESS")
	os.Unsetenv("BASE_URL")
	os.Unsetenv("AUDIT_FILE")
	os.Unsetenv("AUDIT_URL")
	os.Unsetenv("ENABLE_HTTPS")
	os.Unsetenv("TLS_CERT_FILE")
	os.Unsetenv("TLS_KEY_FILE")
	os.Unsetenv("CONFIG")

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.Equal(t, "localhost:8080", cfg.HostAddr)
	assert.Equal(t, "http://localhost:8080", cfg.BaseUrl)
	assert.False(t, cfg.EnableHTTPS)
}

func TestNewConfigEnvOverride(t *testing.T) {
	resetFlags(t)
	t.Setenv("SERVER_ADDRESS", "127.0.0.1:9090")
	t.Setenv("AUDIT_FILE", "/tmp/audit.log")
	t.Setenv("AUDIT_URL", "http://audit.local/log")
	t.Setenv("ENABLE_HTTPS", "true")
	t.Setenv("TLS_CERT_FILE", "/tmp/cert.pem")
	t.Setenv("TLS_KEY_FILE", "/tmp/key.pem")

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9090", cfg.HostAddr)
	assert.Equal(t, "/tmp/audit.log", cfg.AuditFile)
	assert.Equal(t, "http://audit.local/log", cfg.AuditURL)
	assert.True(t, cfg.EnableHTTPS)
	assert.Equal(t, "/tmp/cert.pem", cfg.TLSCertFile)
	assert.Equal(t, "/tmp/key.pem", cfg.TLSKeyFile)
}

func TestNewConfigFromFile(t *testing.T) {
	resetFlags(t)
	t.Setenv("CONFIG", "")
	os.Unsetenv("SERVER_ADDRESS")
	os.Unsetenv("BASE_URL")
	os.Unsetenv("ENABLE_HTTPS")

	configPath := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(configPath, []byte(`{
		"server_address": "127.0.0.1:7070",
		"base_url": "http://127.0.0.1:7070",
		"file_storage_path": "/tmp/storage.db",
		"database_dsn": "postgres://localhost/db",
		"cookie_secret": "secret",
		"audit_file": "/tmp/audit.log",
		"audit_url": "http://audit.local/log",
		"enable_https": true,
		"tls_cert_file": "/tmp/cert.pem",
		"tls_key_file": "/tmp/key.pem"
	}`), 0o600)
	require.NoError(t, err)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"shortener", "-c", configPath}

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:7070", cfg.HostAddr)
	assert.Equal(t, "http://127.0.0.1:7070", cfg.BaseUrl)
	assert.Equal(t, "/tmp/storage.db", cfg.FilePath)
	assert.Equal(t, "postgres://localhost/db", cfg.DatabaseDSN)
	assert.Equal(t, "secret", cfg.CookieSecret)
	assert.Equal(t, "/tmp/audit.log", cfg.AuditFile)
	assert.Equal(t, "http://audit.local/log", cfg.AuditURL)
	assert.True(t, cfg.EnableHTTPS)
	assert.Equal(t, "/tmp/cert.pem", cfg.TLSCertFile)
	assert.Equal(t, "/tmp/key.pem", cfg.TLSKeyFile)
}

func TestNewConfigFileOverriddenByEnv(t *testing.T) {
	resetFlags(t)

	configPath := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(configPath, []byte(`{
		"server_address": "127.0.0.1:7070",
		"base_url": "http://127.0.0.1:7070"
	}`), 0o600)
	require.NoError(t, err)

	t.Setenv("CONFIG", configPath)
	t.Setenv("SERVER_ADDRESS", "127.0.0.1:9090")

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9090", cfg.HostAddr)
	assert.Equal(t, "http://127.0.0.1:7070", cfg.BaseUrl)
}

func TestNewConfigFileOverriddenByFlag(t *testing.T) {
	resetFlags(t)
	os.Unsetenv("SERVER_ADDRESS")

	configPath := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(configPath, []byte(`{
		"server_address": "127.0.0.1:7070",
		"base_url": "http://127.0.0.1:7070"
	}`), 0o600)
	require.NoError(t, err)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"shortener", "-c", configPath, "-a", "127.0.0.1:9090"}

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9090", cfg.HostAddr)
	assert.Equal(t, "http://127.0.0.1:7070", cfg.BaseUrl)
}

func TestNewConfigEnvOverriddenByFlag(t *testing.T) {
	resetFlags(t)
	t.Setenv("SERVER_ADDRESS", "127.0.0.1:9090")
	t.Setenv("BASE_URL", "http://127.0.0.1:9090")
	t.Setenv("ENABLE_HTTPS", "true")

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"shortener", "-a", "127.0.0.1:7070", "-b", "http://127.0.0.1:7070", "-s=false"}

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:7070", cfg.HostAddr)
	assert.Equal(t, "http://127.0.0.1:7070", cfg.BaseUrl)
	assert.False(t, cfg.EnableHTTPS)
}

func TestNewConfigFromEnvConfigPath(t *testing.T) {
	resetFlags(t)
	os.Unsetenv("SERVER_ADDRESS")

	configPath := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(configPath, []byte(`{
		"server_address": "127.0.0.1:7070"
	}`), 0o600)
	require.NoError(t, err)

	t.Setenv("CONFIG", configPath)

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:7070", cfg.HostAddr)
}
