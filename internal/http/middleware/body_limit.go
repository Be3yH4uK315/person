package middleware

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func BodyLimit(maxBytes int64) echo.MiddlewareFunc {
	if maxBytes <= 0 {
		panic("body limit must be positive")
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			request := c.Request()
			if request.Body != nil {
				request.Body = http.MaxBytesReader(c.Response(), request.Body, maxBytes)
			}
			return next(c)
		}
	}
}
