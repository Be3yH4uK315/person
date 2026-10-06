package middleware

import "github.com/labstack/echo/v5"

func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			headers := c.Response().Header()
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("Referrer-Policy", "no-referrer")
			headers.Set("Cache-Control", "no-store")
			return next(c)
		}
	}
}
