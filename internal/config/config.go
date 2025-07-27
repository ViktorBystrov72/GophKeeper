// Package config управляет конфигурацией сервера GophKeeper.
package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

// ServerConfig содержит все параметры конфигурации сервера.
type ServerConfig struct {
	ServerAddress   string
	GRPCAddress     string
	EnableTLS       bool
	CertFile        string
	KeyFile         string
	DatabaseURI     string
	MigrationsPath  string
	JWTSecret       string
	EncryptionKey   string
	LogLevel        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// ClientConfig содержит все параметры конфигурации клиента.
type ClientConfig struct {
	ServerAddress string
	GRPCAddress   string
	EnableTLS     bool
	CertFile      string
	ConfigPath    string
	LogLevel      string
	Timeout       time.Duration
}

// NewServerConfig создает новую конфигурацию сервера со значениями по умолчанию.
func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		ServerAddress:   ":8080",
		GRPCAddress:     ":8081",
		EnableTLS:       false,
		CertFile:        "",
		KeyFile:         "",
		DatabaseURI:     "",
		MigrationsPath:  "./migrations",
		JWTSecret:       "",
		EncryptionKey:   "",
		LogLevel:        "info",
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}
}

// NewClientConfig создает новую конфигурацию клиента со значениями по умолчанию.
func NewClientConfig() *ClientConfig {
	return &ClientConfig{
		ServerAddress: "localhost:8080",
		GRPCAddress:   "localhost:8081",
		EnableTLS:     false,
		CertFile:      "",
		ConfigPath:    "./config.json",
		LogLevel:      "info",
		Timeout:       30 * time.Second,
	}
}

// LoadServerConfig загружает конфигурацию сервера из флагов и переменных окружения.
// Флаги имеют приоритет над переменными окружения.
func LoadServerConfig() (*ServerConfig, error) {
	cfg := NewServerConfig()

	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "Server address")
	flag.StringVar(&cfg.GRPCAddress, "g", cfg.GRPCAddress, "gRPC server address")
	flag.BoolVar(&cfg.EnableTLS, "s", cfg.EnableTLS, "Enable TLS")
	flag.StringVar(&cfg.CertFile, "cert", cfg.CertFile, "TLS certificate file")
	flag.StringVar(&cfg.KeyFile, "key", cfg.KeyFile, "TLS private key file")
	flag.StringVar(&cfg.DatabaseURI, "d", cfg.DatabaseURI, "Database connection string")
	flag.StringVar(&cfg.MigrationsPath, "m", cfg.MigrationsPath, "Migrations directory path")
	flag.StringVar(&cfg.JWTSecret, "jwt", cfg.JWTSecret, "JWT secret key")
	flag.StringVar(&cfg.EncryptionKey, "enc", cfg.EncryptionKey, "Encryption key")
	flag.StringVar(&cfg.LogLevel, "l", cfg.LogLevel, "Log level")

	flag.Parse()

	// Загружаем из переменных окружения если флаги не установлены
	if cfg.ServerAddress == ":8080" {
		if addr := os.Getenv("SERVER_ADDRESS"); addr != "" {
			cfg.ServerAddress = addr
		}
	}

	if cfg.GRPCAddress == ":8081" {
		if addr := os.Getenv("GRPC_ADDRESS"); addr != "" {
			cfg.GRPCAddress = addr
		}
	}

	if !cfg.EnableTLS {
		if tls := os.Getenv("ENABLE_TLS"); tls != "" {
			if val, err := strconv.ParseBool(tls); err == nil {
				cfg.EnableTLS = val
			}
		}
	}

	if cfg.CertFile == "" {
		if cert := os.Getenv("CERT_FILE"); cert != "" {
			cfg.CertFile = cert
		}
	}

	if cfg.KeyFile == "" {
		if key := os.Getenv("KEY_FILE"); key != "" {
			cfg.KeyFile = key
		}
	}

	if cfg.DatabaseURI == "" {
		if uri := os.Getenv("DATABASE_URI"); uri != "" {
			cfg.DatabaseURI = uri
		}
	}

	if cfg.JWTSecret == "" {
		if secret := os.Getenv("JWT_SECRET"); secret != "" {
			cfg.JWTSecret = secret
		}
	}

	if cfg.EncryptionKey == "" {
		if key := os.Getenv("ENCRYPTION_KEY"); key != "" {
			cfg.EncryptionKey = key
		}
	}

	if cfg.LogLevel == "info" {
		if level := os.Getenv("LOG_LEVEL"); level != "" {
			cfg.LogLevel = level
		}
	}

	// Валидируем конфигурацию на раннем этапе
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// LoadClientConfig загружает конфигурацию клиента из флагов и переменных окружения.
func LoadClientConfig() (*ClientConfig, error) {
	cfg := NewClientConfig()

	flag.StringVar(&cfg.ServerAddress, "server", cfg.ServerAddress, "Server address")
	flag.StringVar(&cfg.GRPCAddress, "grpc", cfg.GRPCAddress, "gRPC server address")
	flag.BoolVar(&cfg.EnableTLS, "tls", cfg.EnableTLS, "Enable TLS")
	flag.StringVar(&cfg.CertFile, "cert", cfg.CertFile, "TLS certificate file")
	flag.StringVar(&cfg.ConfigPath, "config", cfg.ConfigPath, "Configuration file path")
	flag.StringVar(&cfg.LogLevel, "log", cfg.LogLevel, "Log level")

	flag.Parse()
	if cfg.ServerAddress == "localhost:8080" {
		if addr := os.Getenv("SERVER_ADDRESS"); addr != "" {
			cfg.ServerAddress = addr
		}
	}

	if cfg.GRPCAddress == "localhost:8081" {
		if addr := os.Getenv("GRPC_ADDRESS"); addr != "" {
			cfg.GRPCAddress = addr
		}
	}

	if !cfg.EnableTLS {
		if tls := os.Getenv("ENABLE_TLS"); tls != "" {
			if val, err := strconv.ParseBool(tls); err == nil {
				cfg.EnableTLS = val
			}
		}
	}

	if cfg.CertFile == "" {
		if cert := os.Getenv("CERT_FILE"); cert != "" {
			cfg.CertFile = cert
		}
	}

	if cfg.ConfigPath == "./config.json" {
		if path := os.Getenv("CONFIG_PATH"); path != "" {
			cfg.ConfigPath = path
		}
	}

	if cfg.LogLevel == "info" {
		if level := os.Getenv("LOG_LEVEL"); level != "" {
			cfg.LogLevel = level
		}
	}

	return cfg, nil
}

// Validate проверяет корректность конфигурации сервера.
func (c *ServerConfig) Validate() error {
	if c.DatabaseURI == "" {
		return fmt.Errorf("database URI is required")
	}

	if c.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	if c.EncryptionKey == "" {
		return fmt.Errorf("encryption key is required")
	}

	if c.EnableTLS {
		if c.CertFile == "" {
			return fmt.Errorf("TLS certificate file is required when TLS is enabled")
		}
		if c.KeyFile == "" {
			return fmt.Errorf("TLS private key file is required when TLS is enabled")
		}
	}

	return nil
}
