package main

import (
	"context"
	"embed"
	"fmt"
	"gophermart/internal/app/app"
	"gophermart/internal/app/config"
	"gophermart/internal/app/logger"
	"net/http"
	"os"
	"os/signal"
	"time"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
	// setting up signal capturing
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		osCall := <-stop
		logger.Global().Info().Str("signal", fmt.Sprintf("%+v", osCall)).Msg("System call")
		cancel()
	}()

	c := config.New()
	if err := c.Load(); err != nil {
		logger.Global().Err(err).Msg("Config load failed")
	}
	c.LogVerbose = true

	if err := runServer(ctx, c); err != nil {
		logger.Global().Fatal().Err(err).Msg("Server run failed")
	}
}

func runServer(ctx context.Context, c config.Config) (err error) {
	l := logger.New(c.LogVerbose, c.LogPretty)

	a, err := app.New(c, l, embedMigrations)
	if err != nil {
		return fmt.Errorf("app init: %w", err)
	}
	defer a.Stop()

	srv := &http.Server{
		Addr:         c.Server.Listen,
		Handler:      a.Router(),
		ReadTimeout:  c.Server.TimeoutRead,
		WriteTimeout: c.Server.TimeoutWrite,
		IdleTimeout:  c.Server.TimeoutIdle,
	}

	go func() {
		l.Info().Str("listen_address", c.Server.Listen).Msg("Listening incoming connections")
		if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Fatal().Err(err).Msg("")
		}
	}()

	l.Info().Msg("Server started")
	<-ctx.Done()
	l.Info().Msg("Server stopped")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err = srv.Shutdown(ctxShutdown); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	l.Printf("Server exited properly")

	return
}
