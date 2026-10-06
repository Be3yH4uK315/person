package middleware

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

type contextTestKey struct{}

func TestContextEnricher_AddsValueToRequestContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	handler := ContextEnricher(func(parent context.Context) context.Context {
		return context.WithValue(parent, contextTestKey{}, "value")
	})(func(c *echo.Context) error {
		require.Equal(t, "value", c.Request().Context().Value(contextTestKey{}))
		return nil
	})

	require.NoError(t, handler(ctx))
}

func TestRecovery_ConvertsPanicToError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	err := Recovery()(func(*echo.Context) error {
		panic("boom")
	})(ctx)

	var panicErr *PanicError
	require.ErrorAs(t, err, &panicErr)
	require.Equal(t, "boom", panicErr.Value)
	require.NotEmpty(t, panicErr.Stack)
}

func TestSecurityHeaders_SetsExpectedHeaders(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	err := SecurityHeaders()(func(*echo.Context) error { return nil })(ctx)

	require.NoError(t, err)
	require.Equal(t, "nosniff", resp.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "no-referrer", resp.Header().Get("Referrer-Policy"))
	require.Equal(t, "no-store", resp.Header().Get("Cache-Control"))
}

func TestBodyLimit_RejectsOversizedBodyWhenRead(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader("123456"))
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	err := BodyLimit(5)(func(c *echo.Context) error {
		_, readErr := io.ReadAll(c.Request().Body)
		return readErr
	})(ctx)

	var maxBytesErr *http.MaxBytesError
	require.True(t, errors.As(err, &maxBytesErr))
}

func runLogger(t *testing.T, logger *logrus.Logger, target string, handler echo.HandlerFunc) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	ctx := e.NewContext(req, httptest.NewRecorder())

	require.NoError(t, Logger(logger)(handler)(ctx))
}

func TestLogger_LogsRequestWithLevelByStatus(t *testing.T) {
	tests := []struct {
		status    int
		wantLevel logrus.Level
	}{
		{http.StatusOK, logrus.InfoLevel},
		{http.StatusNotFound, logrus.WarnLevel},
		{http.StatusInternalServerError, logrus.ErrorLevel},
	}

	for _, tt := range tests {
		logger, hook := logrustest.NewNullLogger()

		runLogger(t, logger, "/person", func(c *echo.Context) error {
			return c.NoContent(tt.status)
		})

		require.Len(t, hook.AllEntries(), 1, tt.status)
		entry := hook.LastEntry()
		require.Equal(t, tt.wantLevel, entry.Level, tt.status)
		require.Equal(t, "http request", entry.Message)
		require.EqualValues(t, tt.status, entry.Data["status"])
		require.Equal(t, http.MethodGet, entry.Data["method"])
	}
}

func TestLogger_SkipsHealthChecks(t *testing.T) {
	logger, hook := logrustest.NewNullLogger()

	runLogger(t, logger, "/health/live", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	require.Empty(t, hook.AllEntries())
}

func TestSkipHealthChecks(t *testing.T) {
	e := echo.New()
	for target, want := range map[string]bool{
		"/health/live":  true,
		"/health/ready": true,
		"/person":       false,
		"/health":       false,
	} {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)

		require.Equal(t, want, SkipHealthChecks(e.NewContext(req, httptest.NewRecorder())), target)
	}
}
