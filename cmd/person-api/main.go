package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

var version = "dev"

func main() {
	logger := logrus.New()
	if err := run(logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.WithError(err).Error("application stopped")
		os.Exit(1)
	}
}

func run(logger *logrus.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := NewApplication(logger)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}
	defer func() {
		if err := application.Shutdown(); err != nil {
			logger.WithError(err).Error("shutdown application")
		}
	}()

	if err := application.Ping(ctx); err != nil {
		return err
	}
	return application.Run(ctx)
}
