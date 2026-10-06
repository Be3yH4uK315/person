package middleware

import (
	"context"

	"github.com/labstack/echo/v5"
)

func ContextEnricher(enrich func(context.Context) context.Context) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ctx := enrich(c.Request().Context())
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
