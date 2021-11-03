package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
	"gophermart/internal/app/apperr"
	"gophermart/internal/app/logger"
	mw "gophermart/internal/app/middleware"
	"gophermart/pkg/accrual"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	// setting up signal capturing
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		osCall := <-stop
		log.Printf("System call: %+v", osCall)
		cancel()
	}()

	l := logger.New(true, true)

	if err := runServer(ctx, "127.0.0.1:8090", l); err != nil {
		l.Fatal().Err(err).Msg("Server run failed")
	}
}

func runServer(ctx context.Context, listenAddr string, l logger.Logger) (err error) {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(mw.Log(l))
	r.Get("/api/orders/{order}", GetOrder)

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: r,
	}

	go func() {
		log.Printf("Listening on %s", listenAddr)
		if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Fatal().Err(err).Msg("")
		}
	}()

	log.Printf("Server started")
	<-ctx.Done()
	log.Printf("Server stopped")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err = srv.Shutdown(ctxShutdown); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Printf("Server exited properly")

	return
}

func GetOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "order")
	ctx := r.Context()
	l := logger.Ctx(ctx).With().Str("order_id", id).Str("method", "GetOrder").Logger()

	if id == "" {
		l.Error().Msg("Empty order id")
		http.Error(w, apperr.ErrNotFound.Error(), http.StatusNotFound)
	}

	out := &accrual.GetOrderResponse{}
	rawJson, _ := json.Marshal(out)

	if rand.Float32() < 0.5 {
		http.Error(w, "fail", http.StatusInternalServerError)
		return
	}

	if rand.Float32() < 0.5 {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}

	_, _ = w.Write(rawJson)
}
