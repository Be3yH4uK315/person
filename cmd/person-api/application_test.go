package main

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func quietLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

func TestNewApplication_RejectsNilLogger(t *testing.T) {
	application, err := NewApplication(nil)

	require.Nil(t, application)
	require.ErrorContains(t, err, "logger is required")
}

func TestNewApplication_FailsWithoutDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	application, err := NewApplication(quietLogger())

	require.Nil(t, application)
	require.ErrorContains(t, err, "DATABASE_URL is required")
}

func TestNewApplication_DoesNotConnectUntilPing(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://person:secret@127.0.0.1:1/person?sslmode=disable")
	t.Setenv("DB_PING_TIMEOUT", "500ms")

	application, err := NewApplication(quietLogger())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, application.Shutdown()) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.ErrorContains(t, application.Ping(ctx), "ping database")
}
