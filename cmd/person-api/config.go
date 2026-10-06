package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type Config struct {
	HTTP     HTTPConfig
	Database DatabaseConfig
	Log      LogConfig
}

type HTTPConfig struct {
	Address           string
	GracefulTimeout   time.Duration
	ContextTimeout    time.Duration
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxBodyBytes      int64
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	PingTimeout     time.Duration
}

type LogConfig struct {
	Level  logrus.Level
	Format string
}

func LoadConfig() (Config, error) {
	var cfg Config
	var err error

	cfg.HTTP.Address = envString("HTTP_ADDRESS", ":8080")
	if cfg.HTTP.GracefulTimeout, err = envDuration("HTTP_GRACEFUL_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.ContextTimeout, err = envDuration("HTTP_CONTEXT_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.ReadHeaderTimeout, err = envDuration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.ReadTimeout, err = envDuration("HTTP_READ_TIMEOUT", 15*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.WriteTimeout, err = envDuration("HTTP_WRITE_TIMEOUT", 15*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.IdleTimeout, err = envDuration("HTTP_IDLE_TIMEOUT", 60*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.MaxBodyBytes, err = envInt64("HTTP_MAX_BODY_BYTES", 1<<20); err != nil {
		return Config{}, err
	}

	cfg.Database.URL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if cfg.Database.URL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.Database.MaxOpenConns, err = envInt("DB_MAX_OPEN_CONNS", 10); err != nil {
		return Config{}, err
	}
	if cfg.Database.MaxIdleConns, err = envInt("DB_MAX_IDLE_CONNS", 5); err != nil {
		return Config{}, err
	}
	if cfg.Database.ConnMaxLifetime, err = envDuration("DB_CONN_MAX_LIFETIME", time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.Database.ConnMaxIdleTime, err = envDuration("DB_CONN_MAX_IDLE_TIME", 15*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.Database.PingTimeout, err = envDuration("DB_PING_TIMEOUT", 5*time.Second); err != nil {
		return Config{}, err
	}

	level := envString("LOG_LEVEL", "info")
	cfg.Log.Level, err = logrus.ParseLevel(level)
	if err != nil {
		return Config{}, fmt.Errorf("LOG_LEVEL: %w", err)
	}
	cfg.Log.Format = strings.ToLower(envString("LOG_FORMAT", "json"))

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

const (
	maxHTTPBodyBytes = 10 << 20
	maxDBOpenConns   = 100
)

func (c Config) validate() error {
	if strings.TrimSpace(c.HTTP.Address) == "" {
		return fmt.Errorf("HTTP_ADDRESS must not be empty")
	}
	if c.HTTP.GracefulTimeout <= 0 || c.HTTP.ContextTimeout <= 0 ||
		c.HTTP.ReadHeaderTimeout <= 0 || c.HTTP.ReadTimeout <= 0 ||
		c.HTTP.WriteTimeout <= 0 || c.HTTP.IdleTimeout <= 0 {
		return fmt.Errorf("HTTP timeouts must be positive")
	}
	if c.HTTP.ContextTimeout >= c.HTTP.WriteTimeout {
		return fmt.Errorf("HTTP_CONTEXT_TIMEOUT must be less than HTTP_WRITE_TIMEOUT")
	}
	if c.HTTP.MaxBodyBytes <= 0 || c.HTTP.MaxBodyBytes > maxHTTPBodyBytes {
		return fmt.Errorf("HTTP_MAX_BODY_BYTES must be between 1 and %d", maxHTTPBodyBytes)
	}
	if c.Database.MaxOpenConns <= 0 || c.Database.MaxOpenConns > maxDBOpenConns {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be between 1 and %d", maxDBOpenConns)
	}

	if c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must be zero or greater")
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must not exceed DB_MAX_OPEN_CONNS")
	}
	if c.Database.ConnMaxLifetime <= 0 || c.Database.ConnMaxIdleTime <= 0 || c.Database.PingTimeout <= 0 {
		return fmt.Errorf("database timeouts must be positive")
	}
	if c.Log.Format != "json" && c.Log.Format != "text" {
		return fmt.Errorf("LOG_FORMAT must be json or text")
	}
	return nil
}

func envString(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return parsed, nil
}

func envInt64(name string, fallback int64) (int64, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return parsed, nil
}

func envDuration(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", name, err)
	}
	return parsed, nil
}
