package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"person/internal/app"
)

func TestClassifyError_MapsDomainValidationToHTTPValidation(t *testing.T) {
	_, err := app.NewPerson(app.PersonDetails{})
	require.Error(t, err)

	status, code, message, details := classifyError(err)

	require.Equal(t, http.StatusBadRequest, status)
	require.Equal(t, "VALIDATION_ERROR", code)
	require.Equal(t, "invalid request", message)
	require.Equal(t, map[string]string{
		"email":     "is required",
		"phone":     "is required",
		"firstName": "is required",
		"lastName":  "is required",
	}, details)
}

func TestClassifyError_MapsApplicationValidationToHTTPValidation(t *testing.T) {
	err := app.NewValidationError(map[string]string{"version": "must be positive"})

	status, code, message, details := classifyError(err)

	require.Equal(t, http.StatusBadRequest, status)
	require.Equal(t, "VALIDATION_ERROR", code)
	require.Equal(t, "invalid request", message)
	require.Equal(t, map[string]string{"version": "must be positive"}, details)
}

func TestClassifyError_MapsHTTPStatusWithoutCollapsingTo400(t *testing.T) {
	for _, tc := range []struct {
		status  int
		code    string
		message string
	}{
		{http.StatusUnauthorized, "UNAUTHORIZED", "authentication required"},
		{http.StatusForbidden, "FORBIDDEN", "access denied"},
		{http.StatusTooManyRequests, "RATE_LIMITED", "too many requests"},
	} {
		status, code, message, details := classifyError(echo.NewHTTPError(tc.status, "test"))
		require.Equal(t, tc.status, status)
		require.Equal(t, tc.code, code)
		require.Equal(t, tc.message, message)
		require.Nil(t, details)
	}
}

func TestClassifyError_MapsEchoRoutingErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "not found", err: echo.ErrNotFound, status: http.StatusNotFound, code: "ROUTE_NOT_FOUND"},
		{name: "method not allowed", err: echo.ErrMethodNotAllowed, status: http.StatusMethodNotAllowed, code: "METHOD_NOT_ALLOWED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, code, message, details := classifyError(tt.err)
			require.Equal(t, tt.status, status)
			require.Equal(t, tt.code, code)
			require.NotEmpty(t, message)
			require.Nil(t, details)
		})
	}
}

func TestClassifyError_MapsCorruptedPersonToInternalError(t *testing.T) {
	err := app.ErrCorruptedPerson

	status, code, message, details := classifyError(err)

	require.Equal(t, http.StatusInternalServerError, status)
	require.Equal(t, "INTERNAL_ERROR", code)
	require.Equal(t, "internal server error", message)
	require.Nil(t, details)
}

func TestClassifyError_MapsInvalidPersonStateToInternalError(t *testing.T) {
	status, code, message, details := classifyError(app.ErrInvalidPersonState)

	require.Equal(t, http.StatusInternalServerError, status)
	require.Equal(t, "INTERNAL_ERROR", code)
	require.Equal(t, "internal server error", message)
	require.Nil(t, details)
}

func TestHTTPErrorHandler_DoesNotRewriteCommittedResponse(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/person", nil)
	resp := httptest.NewRecorder()
	ctx := e.NewContext(req, resp)

	require.NoError(t, ctx.String(http.StatusOK, "already sent"))
	NewHTTPErrorHandler()(ctx, echo.NewHTTPError(http.StatusInternalServerError, "test"))

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "already sent", resp.Body.String())
}

func TestClassifyError_MapsContextTimeout(t *testing.T) {
	status, code, message, details := classifyError(context.DeadlineExceeded)

	require.Equal(t, http.StatusGatewayTimeout, status)
	require.Equal(t, "REQUEST_TIMEOUT", code)
	require.Equal(t, "request timed out", message)
	require.Nil(t, details)
}
