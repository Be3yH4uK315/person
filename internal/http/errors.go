package http

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"person/internal/app"
	"person/internal/http/requestcontext"
	"person/internal/http/response"
	"person/internal/logic"
)

func NewHTTPErrorHandler() func(c *echo.Context, err error) {
	return func(c *echo.Context, err error) {
		if response, unwrapErr := echo.UnwrapResponse(c.Response()); unwrapErr == nil && response.Committed {
			return
		}

		status, code, message, details := classifyError(err)
		requestID := requestcontext.RequestID(c.Request().Context())

		returnError := response.ErrorResponse{
			Error: response.ErrorBody{
				Code:    code,
				Message: message,
				Details: details,
			},
			RequestID: requestID,
		}
		_ = c.JSON(status, returnError)
	}
}

func classifyError(err error) (int, string, string, map[string]string) {
	if err == nil {
		return http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil
	}

	var validation *app.ValidationError
	if errors.As(err, &validation) {
		return http.StatusBadRequest, "VALIDATION_ERROR", "invalid request", validation.Fields
	}

	switch {
	case errors.Is(err, logic.ErrPersonNotFound):
		return http.StatusNotFound, "PERSON_NOT_FOUND", "person not found", nil
	case errors.Is(err, logic.ErrConcurrencyConflict):
		return http.StatusConflict, "CONCURRENCY_CONFLICT", "person version conflict", nil
	case errors.Is(err, logic.ErrEmailAlreadyExists):
		return http.StatusConflict, "PERSON_EMAIL_ALREADY_EXISTS", "person with this email already exists", nil
	case errors.Is(err, app.ErrCorruptedPerson), errors.Is(err, app.ErrInvalidPersonState):
		return http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout, "REQUEST_TIMEOUT", "request timed out", nil
	case errors.Is(err, context.Canceled):
		return 499, "CLIENT_CLOSED_REQUEST", "client closed the request", nil
	}

	var statusCoder echo.HTTPStatusCoder
	if errors.As(err, &statusCoder) {
		return classifyHTTPError(statusCoder.StatusCode())
	}

	return http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil
}

func classifyHTTPError(status int) (int, string, string, map[string]string) {
	if status < 400 || status > 599 {
		return http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil
	}

	messages := map[int]struct {
		code    string
		message string
	}{
		http.StatusBadRequest:            {"BAD_REQUEST", "invalid request"},
		http.StatusUnauthorized:          {"UNAUTHORIZED", "authentication required"},
		http.StatusForbidden:             {"FORBIDDEN", "access denied"},
		http.StatusNotFound:              {"ROUTE_NOT_FOUND", "route not found"},
		http.StatusMethodNotAllowed:      {"METHOD_NOT_ALLOWED", "method not allowed"},
		http.StatusRequestTimeout:        {"REQUEST_TIMEOUT", "request timed out"},
		http.StatusConflict:              {"CONFLICT", "request conflicts with current state"},
		http.StatusRequestEntityTooLarge: {"PAYLOAD_TOO_LARGE", "request body too large"},
		http.StatusUnsupportedMediaType:  {"UNSUPPORTED_MEDIA_TYPE", "unsupported media type"},
		http.StatusUnprocessableEntity:   {"UNPROCESSABLE_ENTITY", "request cannot be processed"},
		http.StatusTooManyRequests:       {"RATE_LIMITED", "too many requests"},
		http.StatusInternalServerError:   {"INTERNAL_ERROR", "internal server error"},
		http.StatusBadGateway:            {"BAD_GATEWAY", "bad gateway"},
		http.StatusServiceUnavailable:    {"SERVICE_UNAVAILABLE", "service unavailable"},
		http.StatusGatewayTimeout:        {"GATEWAY_TIMEOUT", "upstream service timed out"},
	}
	if item, ok := messages[status]; ok {
		return status, item.code, item.message, nil
	}

	message := strings.ToLower(http.StatusText(status))
	if message == "" {
		message = "http request failed"
	}
	return status, "HTTP_ERROR", message, nil
}
