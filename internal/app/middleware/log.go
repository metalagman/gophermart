package middleware

import (
	"github.com/justinas/alice"
	"github.com/rs/zerolog/hlog"
	"gophermart/internal/app/logger"
	"gophermart/internal/app/middleware/trace"
	"net/http"
	"time"
)

func Log(l logger.Logger) func(next http.Handler) http.Handler {
	l = l.Component("Middleware::Log")
	return func(next http.Handler) http.Handler {
		c := alice.New()

		// Install the logger handler with default output on the console
		c = c.Append(hlog.NewHandler(l.Logger))

		// Install some provided extra handler to set some request's context fields.
		// Thanks to that handler, all our logs will come with some prepopulated fields.
		c = c.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
			hlog.FromRequest(r).Info().
				Str("method", r.Method).
				Stringer("url", r.URL).
				Int("status", status).
				Int("size", size).
				Dur("duration", duration).
				Msg("")
		}))
		c = c.Append(hlog.RemoteAddrHandler("ip"))
		c = c.Append(hlog.UserAgentHandler("user_agent"))
		c = c.Append(hlog.RefererHandler("referer"))
		c = c.Append(hlog.RequestIDHandler("request_id", "Request-Id"))
		c = c.Append(trace.CorrelationIDHandler("correlation_id", "X-Correlation-Id"))

		// Here is your final handler
		h := c.Then(next)

		return h
	}
}
