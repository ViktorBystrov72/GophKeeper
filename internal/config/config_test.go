package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewServerConfig(t *testing.T) {
	cfg := NewServerConfig()
	require.NotNil(t, cfg)
	require.Equal(t, ":8080", cfg.ServerAddress)
	require.Equal(t, ":8081", cfg.GRPCAddress)
	require.False(t, cfg.EnableTLS)
	require.Equal(t, "info", cfg.LogLevel)
}

func TestNewClientConfig(t *testing.T) {
	cfg := NewClientConfig()
	require.NotNil(t, cfg)
	require.Equal(t, "localhost:8080", cfg.ServerAddress)
	require.Equal(t, "localhost:8081", cfg.GRPCAddress)
	require.False(t, cfg.EnableTLS)
	require.Equal(t, "info", cfg.LogLevel)
}

func TestServerConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ServerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &ServerConfig{
				DatabaseURI:   "postgres://user:pass@localhost/db",
				JWTSecret:     "secret",
				EncryptionKey: "key",
			},
			wantErr: false,
		},
		{
			name: "missing database URI",
			cfg: &ServerConfig{
				JWTSecret:     "secret",
				EncryptionKey: "key",
			},
			wantErr: true,
		},
		{
			name: "missing JWT secret",
			cfg: &ServerConfig{
				DatabaseURI:   "postgres://user:pass@localhost/db",
				EncryptionKey: "key",
			},
			wantErr: true,
		},
		{
			name: "missing encryption key",
			cfg: &ServerConfig{
				DatabaseURI: "postgres://user:pass@localhost/db",
				JWTSecret:   "secret",
			},
			wantErr: true,
		},
		{
			name: "TLS enabled without cert",
			cfg: &ServerConfig{
				DatabaseURI:   "postgres://user:pass@localhost/db",
				JWTSecret:     "secret",
				EncryptionKey: "key",
				EnableTLS:     true,
			},
			wantErr: true,
		},
		{
			name: "TLS enabled without key",
			cfg: &ServerConfig{
				DatabaseURI:   "postgres://user:pass@localhost/db",
				JWTSecret:     "secret",
				EncryptionKey: "key",
				EnableTLS:     true,
				CertFile:      "cert.pem",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
