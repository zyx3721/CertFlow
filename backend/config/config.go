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

// Load 从环境变量加载配置。
func Load(logger *slog.Logger) (Config, error) {
	return LoadWithOverrides(nil, logger)
}

// LoadWithOverrides 加载配置，overrides 中显式提供的键优先于环境变量，键名与环境变量保持一致。
func LoadWithOverrides(overrides map[string]string, logger *slog.Logger) (Config, error) {
	lookup := func(key, fallback string) string {
		if value, ok := overrides[key]; ok && strings.TrimSpace(value) != "" {
			return value
		}
		return env(key, fallback)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(lookup("PKI_KEY_ENCRYPTION_KEY", "")))
	if err != nil || len(key) != 32 {
		return Config{}, fmt.Errorf("PKI_KEY_ENCRYPTION_KEY must be a base64 encoded 32-byte key")
	}
	cfg := Config{
		Server: ServerConfig{
			Host: lookup("SERVER_HOST", "127.0.0.1"),
			Port: lookup("SERVER_PORT", "8080"),
			Mode: lookup("SERVER_MODE", "release"),
		},
		Database: DatabaseConfig{
			Host:     lookup("DB_HOST", "localhost"),
			Port:     lookup("DB_PORT", "5432"),
			Name:     lookup("DB_NAME", "certflow"),
			User:     lookup("DB_USER", "certflow"),
			Password: lookup("DB_PASSWORD", "certflow_dev"),
			SSLMode:  lookup("DB_SSLMODE", "disable"),
		},
		Auth: AuthConfig{
			SessionSecret:      lookup("JWT_SECRET", ""),
			SessionExpireHours: positiveInt(lookup, "JWT_EXPIRE_HOURS", 12),
		},
		PKI: PKIConfig{
			KeyEncryptionKey: key,
		},
		CORS: CORSConfig{Origin: lookup("CORS_ORIGIN", "http://localhost:5173")},
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

func positiveInt(lookup func(string, string) string, key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(lookup(key, "")))
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
