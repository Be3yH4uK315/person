package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
	"github.com/sirupsen/logrus"

	"person/internal/http/middleware"
)

type RouterOptions struct {
	MaxBodyBytes       int64
	ContextTimeout     time.Duration
	HealthCheckTimeout time.Duration
	ContextEnricher    func(context.Context) context.Context
	ReadinessCheck     func(context.Context) error
}

func NewRouter(handler *Handler, logger *logrus.Logger, options RouterOptions) (*echo.Echo, error) {
	if handler == nil {
		return nil, fmt.Errorf("handler is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if options.MaxBodyBytes <= 0 {
		return nil, fmt.Errorf("body limit must be positive")
	}
	if options.ContextTimeout <= 0 {
		return nil, fmt.Errorf("context timeout must be positive")
	}
	if options.HealthCheckTimeout <= 0 {
		return nil, fmt.Errorf("health check timeout must be positive")
	}
	if options.ContextEnricher == nil {
		return nil, fmt.Errorf("context enricher is required")
	}
	if options.ReadinessCheck == nil {
		return nil, fmt.Errorf("readiness check is required")
	}

	e := echo.NewWithConfig(echo.Config{
		HTTPErrorHandler:             NewHTTPErrorHandler(),
		NoGroupAutoRegister404Routes: true,
	})
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger(logger))
	e.Use(middleware.Recovery())
	e.Use(middleware.SecurityHeaders())
	e.Use(echoMiddleware.ContextTimeoutWithConfig(echoMiddleware.ContextTimeoutConfig{
		Timeout: options.ContextTimeout,
		Skipper: middleware.SkipHealthChecks,
	}))

	e.GET("/health/live", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.GET("/health/ready", func(c *echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), options.HealthCheckTimeout)
		defer cancel()
		if err := options.ReadinessCheck(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	api := e.Group("")
	api.Use(middleware.BodyLimit(options.MaxBodyBytes))
	api.Use(middleware.ContextEnricher(options.ContextEnricher))

	api.POST("/person", handler.Create)
	api.POST("/person/", handler.Create)
	api.GET("/person", handler.List)
	api.GET("/person/", handler.List)
	api.GET("/person/:id", handler.Get)
	api.PUT("/person/:id", handler.Update)
	api.DELETE("/person/:id", handler.Delete)

	return e, nil
}
