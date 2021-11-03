package middleware

import (
	"context"
	"gophermart/internal/app/apperr"
	"gophermart/internal/app/handler"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/session"
	"net/http"
	"strings"
)

func Auth(jwt session.Reader) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.Get(r.Context(), "Middleware.Auth")

			reqHeader := r.Header.Get("Authorization")
			splitToken := strings.Split(reqHeader, "Bearer ")
			if len(splitToken) != 2 {
				log.Debug().Str("header", reqHeader).Msg("Invalid Authorization header")
				handler.WriteError(w, apperr.ErrUnauthorized, http.StatusUnauthorized)
				return
			}

			u, err := jwt.Read(r.Context(), splitToken[1])
			if err != nil {
				log.Debug().Err(err).Msg("Token validation failed")
				handler.WriteError(w, apperr.ErrUnauthorized, http.StatusUnauthorized)
				return
			}

			log.Debug().Str("user", u.Name).Msg("User authorized")
			r = r.WithContext(context.WithValue(r.Context(), handler.ContextKeyUser{}, u))
			next.ServeHTTP(w, r)
		})
	}
}
