package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gophermart/internal/app/handler"
	middleware2 "gophermart/internal/app/middleware"
	"net/http"
)

func (a *App) Router() http.Handler {

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware2.Log(a.logger))

	//auth := mw.Auth(a.session)

	// api
	uh := handler.NewUserHandler(a.users, a.session)

	r.Route("/user", func(r chi.Router) {
		r.Post("/login", uh.Login)
		r.Post("/register", uh.Register)
	})

	return r
}
