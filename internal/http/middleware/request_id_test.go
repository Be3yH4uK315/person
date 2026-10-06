package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"person/internal/http/requestcontext"
)

func TestRequestID_GeneratesWhenHeaderIsMissing(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	middleware := RequestID()(func(c *echo.Context) error {
		require.NotEmpty(t, requestcontext.RequestID(c.Request().Context()))
		return nil
	})

	require.NoError(t, middleware(ctx))
	requestID := resp.Header().Get(echo.HeaderXRequestID)
	require.NotEmpty(t, requestID)
	require.NotEmpty(t, requestcontext.RequestID(ctx.Request().Context()))
}

func TestRequestID_PreservesValidUUID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set(echo.HeaderXRequestID, "550e8400-e29b-41d4-a716-446655440000")
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	middleware := RequestID()(func(c *echo.Context) error { return nil })
	require.NoError(t, middleware(ctx))
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", resp.Header().Get(echo.HeaderXRequestID))
}

func TestRequestID_ReplacesInvalidUUID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set(echo.HeaderXRequestID, "not-a-uuid")
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	middleware := RequestID()(func(c *echo.Context) error {
		require.NotEqual(t, "not-a-uuid", requestcontext.RequestID(c.Request().Context()))
		return nil
	})
	require.NoError(t, middleware(ctx))
	require.NotEqual(t, "not-a-uuid", resp.Header().Get(echo.HeaderXRequestID))
}

func TestRequestID_PreservesNginxStyleHexID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set(echo.HeaderXRequestID, "0123456789abcdef0123456789abcdef")
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	require.NoError(t, RequestID()(func(c *echo.Context) error { return nil })(ctx))
	require.Equal(t, "0123456789abcdef0123456789abcdef", requestcontext.RequestID(ctx.Request().Context()))
	require.Equal(t, "0123456789abcdef0123456789abcdef", resp.Header().Get(echo.HeaderXRequestID))
}
