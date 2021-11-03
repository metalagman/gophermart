package app

import (
	"database/sql"
	"fmt"
	"gophermart/internal/app/config"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/session"
	"gophermart/internal/app/storage"
	"gophermart/internal/app/storage/postgres"
	"gophermart/pkg/accrual"
)

type App struct {
	config  config.Config
	logger  logger.Logger
	accrual *accrual.Service
	users   storage.UserRepository
	session session.Manager
	stopCh  chan struct{}
}

func New(cfg config.Config, logger logger.Logger) (*App, error) {
	as, err := accrual.NewService(cfg.Accrual.RemoteURL)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("postgres", cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}

	users, err := postgres.NewUserRepository(db)
	if err != nil {
		return nil, fmt.Errorf("user repository init: %w", err)
	}

	a := &App{
		config:  cfg,
		logger:  logger,
		stopCh:  make(chan struct{}),
		users:   users,
		session: session.NewMemory(cfg.SecretKey, users),
		accrual: as,
	}

	go func() {
		<-a.stopCh
		a.logger.Info().Msg("Shutting down application")
	}()

	return a, nil
}

func (a *App) Stop() {
	close(a.stopCh)
}
