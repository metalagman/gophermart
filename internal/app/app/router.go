package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gophermart/internal/app/handler"
	mw "gophermart/internal/app/middleware"
	"net/http"
)

func (a *App) Router() http.Handler {

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(mw.Log(a.logger))

	auth := mw.Auth(a.session)

	// api
	uh := handler.NewUserHandler(a.users, a.session)
	oh := handler.NewOrderHandler(a.orders)

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/login", uh.Login)
		r.Post("/register", uh.Register)
		r.With(auth).Post("/orders", oh.Create)
		r.With(auth).Get("/orders", oh.List)
	})

	return r
}