package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigDefaults(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Unsetenv("SERVER_ADDRESS")
	os.Unsetenv("BASE_URL")
	os.Unsetenv("AUDIT_FILE")
	os.Unsetenv("AUDIT_URL")
	os.Unsetenv("ENABLE_HTTPS")
	os.Unsetenv("TLS_CERT_FILE")
	os.Unsetenv("TLS_KEY_FILE")

	cfg := NewConfig()
	assert.Equal(t, "localhost:8080", cfg.HostAddr)
	assert.Equal(t, "http://localhost:8080", cfg.BaseUrl)
	assert.False(t, cfg.EnableHTTPS)
}

func TestNewConfigEnvOverride(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	t.Setenv("SERVER_ADDRESS", "127.0.0.1:9090")
	t.Setenv("AUDIT_FILE", "/tmp/audit.log")
	t.Setenv("AUDIT_URL", "http://audit.local/log")
	t.Setenv("ENABLE_HTTPS", "true")
	t.Setenv("TLS_CERT_FILE", "/tmp/cert.pem")
	t.Setenv("TLS_KEY_FILE", "/tmp/key.pem")

	cfg := NewConfig()
	assert.Equal(t, "127.0.0.1:9090", cfg.HostAddr)
	assert.Equal(t, "/tmp/audit.log", cfg.AuditFile)
	assert.Equal(t, "http://audit.local/log", cfg.AuditURL)
	assert.True(t, cfg.EnableHTTPS)
	assert.Equal(t, "/tmp/cert.pem", cfg.TLSCertFile)
	assert.Equal(t, "/tmp/key.pem", cfg.TLSKeyFile)
}
