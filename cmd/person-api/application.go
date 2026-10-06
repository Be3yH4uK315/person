package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gocraft/dbr/v2"
	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"

	personhttp "person/internal/http"
	"person/internal/logic/commands"
	"person/internal/logic/queries"
	"person/internal/postgres"
	"person/internal/postgres/readers"
	"person/internal/postgres/repository"
)

type Application struct {
	server *echo.Echo
	db     *dbr.Connection
	config Config
}

func NewApplication(logger *logrus.Logger) (*Application, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger.SetOutput(os.Stdout)
	if config.Log.Format == "text" {
		logger.SetFormatter(&logrus.TextFormatter{})
	} else {
		logger.SetFormatter(&logrus.JSONFormatter{})
	}
	logger.SetLevel(config.Log.Level)
	logger.WithField("version", version).Info("application initialized")

	connection, err := dbr.Open("pgx", config.Database.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	connection.SetMaxOpenConns(config.Database.MaxOpenConns)
	connection.SetMaxIdleConns(config.Database.MaxIdleConns)
	connection.SetConnMaxLifetime(config.Database.ConnMaxLifetime)
	connection.SetConnMaxIdleTime(config.Database.ConnMaxIdleTime)

	personRepo := repository.NewPersonRepository()
	reader := readers.NewPersonReader()
	uow := postgres.NewUnitOfWork()

	handler := personhttp.NewHandler(
		commands.NewCreatePersonHandler(personRepo, uow),
		commands.NewUpdatePersonHandler(personRepo, uow),
		commands.NewDeletePersonHandler(personRepo, uow),
		queries.NewGetPersonQueryHandler(reader),
		queries.NewListPersonsQueryHandler(reader),
	)

	server, err := personhttp.NewRouter(handler, logger, personhttp.RouterOptions{
		MaxBodyBytes:       config.HTTP.MaxBodyBytes,
		ContextTimeout:     config.HTTP.ContextTimeout,
		HealthCheckTimeout: config.Database.PingTimeout,
		ContextEnricher: func(ctx context.Context) context.Context {
			return postgres.WithSession(ctx, connection.NewSession(nil))
		},
		ReadinessCheck: func(ctx context.Context) error {
			return connection.PingContext(ctx)
		},
	})
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("create router: %w", err)
	}

	return &Application{server: server, db: connection, config: config}, nil
}

func (a *Application) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, a.config.Database.PingTimeout)
	defer cancel()
	if err := a.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

func (a *Application) Run(ctx context.Context) error {
	return echo.StartConfig{
		Address:         a.config.HTTP.Address,
		GracefulTimeout: a.config.HTTP.GracefulTimeout,
		HideBanner:      true,
		BeforeServeFunc: func(server *http.Server) error {
			server.ReadHeaderTimeout = a.config.HTTP.ReadHeaderTimeout
			server.ReadTimeout = a.config.HTTP.ReadTimeout
			server.WriteTimeout = a.config.HTTP.WriteTimeout
			server.IdleTimeout = a.config.HTTP.IdleTimeout
			return nil
		},
	}.Start(ctx, a.server)
}

func (a *Application) Shutdown() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}
