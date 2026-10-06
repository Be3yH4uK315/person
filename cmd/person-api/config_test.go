package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadConfigRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := LoadConfig()
	require.ErrorContains(t, err, "DATABASE_URL is required")
}

func TestLoadConfigReadsEnvironment(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://person:secret@localhost:5432/person?sslmode=disable")
	t.Setenv("HTTP_ADDRESS", ":9090")
	t.Setenv("HTTP_GRACEFUL_TIMEOUT", "20s")
	t.Setenv("HTTP_CONTEXT_TIMEOUT", "12s")
	t.Setenv("HTTP_MAX_BODY_BYTES", "2048")
	t.Setenv("DB_MAX_OPEN_CONNS", "20")
	t.Setenv("DB_MAX_IDLE_CONNS", "8")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("LOG_FORMAT", "text")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, ":9090", cfg.HTTP.Address)
	require.Equal(t, 20*time.Second, cfg.HTTP.GracefulTimeout)
	require.Equal(t, 12*time.Second, cfg.HTTP.ContextTimeout)
	require.EqualValues(t, 2048, cfg.HTTP.MaxBodyBytes)
	require.Equal(t, 20, cfg.Database.MaxOpenConns)
	require.Equal(t, 8, cfg.Database.MaxIdleConns)
	require.Equal(t, "warning", cfg.Log.Level.String())
	require.Equal(t, "text", cfg.Log.Format)
}

func TestLoadConfigRejectsInvalidPoolConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://person:secret@localhost:5432/person?sslmode=disable")
	t.Setenv("DB_MAX_OPEN_CONNS", "5")
	t.Setenv("DB_MAX_IDLE_CONNS", "6")

	_, err := LoadConfig()
	require.ErrorContains(t, err, "DB_MAX_IDLE_CONNS must not exceed DB_MAX_OPEN_CONNS")
}

func TestLoadConfigRejectsInvalidLogFormat(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://person:secret@localhost:5432/person?sslmode=disable")
	t.Setenv("LOG_FORMAT", "xml")

	_, err := LoadConfig()
	require.ErrorContains(t, err, "LOG_FORMAT must be json or text")
}

func TestLoadConfigRejectsContextTimeoutNotLessThanWriteTimeout(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://person:secret@localhost:5432/person?sslmode=disable")
	t.Setenv("HTTP_CONTEXT_TIMEOUT", "15s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "15s")

	_, err := LoadConfig()

	require.ErrorContains(t, err, "HTTP_CONTEXT_TIMEOUT must be less than HTTP_WRITE_TIMEOUT")
}
