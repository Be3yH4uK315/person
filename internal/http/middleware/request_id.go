package middleware

import (
	"regexp"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"person/internal/http/requestcontext"
)

var requestIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			requestID := c.Request().Header.Get(echo.HeaderXRequestID)
			if !isAcceptedRequestID(requestID) {
				requestID = uuid.NewString()
			}

			c.Response().Header().Set(echo.HeaderXRequestID, requestID)
			ctx := requestcontext.WithRequestID(c.Request().Context(), requestID)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

func isAcceptedRequestID(requestID string) bool {
	if _, err := uuid.Parse(requestID); err == nil {
		return true
	}
	return requestIDPattern.MatchString(requestID)
}
