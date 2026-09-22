package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	PKI      PKIConfig
	CORS     CORSConfig
}
type ServerConfig struct{ Host, Port, Mode string }
type DatabaseConfig struct{ Host, Port, Name, User, Password, SSLMode string }
type AuthConfig struct {
	SessionSecret      string
	SessionExpireHours int
}
type PKIConfig struct{ KeyEncryptionKey []byte }
type CORSConfig struct{ Origin string }

func Load(logger *slog.Logger) (Config, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(os.Getenv("PKI_KEY_ENCRYPTION_KEY")))
	if err != nil || len(key) != 32 {
		return Config{}, fmt.Errorf("PKI_KEY_ENCRYPTION_KEY must be a base64 encoded 32-byte key")
	}
	cfg := Config{
		Server: ServerConfig{
			Host: env("SERVER_HOST", "127.0.0.1"),
			Port: env("SERVER_PORT", "8080"),
			Mode: env("SERVER_MODE", "release"),
		},
		Database: DatabaseConfig{
			Host:     env("DB_HOST", "localhost"),
			Port:     env("DB_PORT", "5432"),
			Name:     env("DB_NAME", "certflow"),
			User:     env("DB_USER", "certflow"),
			Password: env("DB_PASSWORD", "certflow_dev"),
			SSLMode:  env("DB_SSLMODE", "disable"),
		},
		Auth: AuthConfig{
			SessionSecret:      os.Getenv("JWT_SECRET"),
			SessionExpireHours: positiveInt("JWT_EXPIRE_HOURS", 12),
		},
		PKI: PKIConfig{
			KeyEncryptionKey: key,
		},
		CORS: CORSConfig{Origin: env("CORS_ORIGIN", "http://localhost:5173")},
	}
	if err := cfg.Database.Validate(); err != nil {
		return Config{}, err
	}
	if cfg.Auth.SessionSecret == "" {
		secret, err := randomSecret(32)
		if err != nil {
			return Config{}, fmt.Errorf("generate session secret: %w", err)
		}
		cfg.Auth.SessionSecret = secret
		logger.Warn("JWT_SECRET is not set; generated a temporary secret for this process")
	}
	return cfg, nil
}
func (s ServerConfig) Addr() string { return net.JoinHostPort(s.Host, s.Port) }
func (d DatabaseConfig) Validate() error {
	if d.Host == "" || d.Port == "" || d.Name == "" || d.User == "" {
		return fmt.Errorf("database host, port, name and user are required")
	}
	return nil
}
func (d DatabaseConfig) DSN() string {
	values := url.Values{}
	values.Set("sslmode", d.SSLMode)
	return (&url.URL{Scheme: "postgres", User: url.UserPassword(d.User, d.Password), Host: net.JoinHostPort(d.Host, d.Port), Path: d.Name, RawQuery: values.Encode()}).String()
}
func (a AuthConfig) SessionTTL() time.Duration {
	return time.Duration(a.SessionExpireHours) * time.Hour
}
func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func positiveInt(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func randomSecret(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
