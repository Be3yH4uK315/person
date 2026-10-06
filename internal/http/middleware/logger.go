package middleware

import (
	"errors"

	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
	"github.com/sirupsen/logrus"
)

func Logger(logger *logrus.Logger) echo.MiddlewareFunc {
	return echoMiddleware.RequestLoggerWithConfig(echoMiddleware.RequestLoggerConfig{
		Skipper:         SkipHealthChecks,
		LogMethod:       true,
		LogRoutePath:    true,
		LogStatus:       true,
		LogLatency:      true,
		LogRemoteIP:     true,
		LogUserAgent:    true,
		LogResponseSize: true,
		LogRequestID:    true,
		HandleError:     true,
		LogValuesFunc: func(_ *echo.Context, values echoMiddleware.RequestLoggerValues) error {
			return logValues(logger, values)
		},
	})
}

func logValues(logger *logrus.Logger, values echoMiddleware.RequestLoggerValues) error {
	routePath := values.RoutePath
	if routePath == "" {
		routePath = values.URIPath
	}

	entry := logger.WithFields(logrus.Fields{
		"request_id":    values.RequestID,
		"method":        values.Method,
		"route":         routePath,
		"status":        values.Status,
		"duration_ms":   float64(values.Latency.Microseconds()) / 1000,
		"remote_ip":     values.RemoteIP,
		"user_agent":    values.UserAgent,
		"response_size": values.ResponseSize,
	})

	if values.Error != nil {
		entry = entry.WithError(values.Error)
		var panicErr *PanicError
		if errors.As(values.Error, &panicErr) {
			entry = entry.WithField("panic", panicErr.Value).WithField("stack", string(panicErr.Stack))
		}
	}

	if values.Status >= 500 {
		entry.Error("http request")
	} else if values.Status >= 400 {
		entry.Warn("http request")
	} else {
		entry.Info("http request")
	}
	return nil
}
