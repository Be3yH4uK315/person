package middleware

import "github.com/labstack/echo/v5"

func SkipHealthChecks(c *echo.Context) bool {
	path := c.Request().URL.Path
	return path == "/health/live" || path == "/health/ready"
}
