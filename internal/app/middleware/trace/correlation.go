package trace

import (
	"context"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"net/http"
)

type cidKey struct{}

// IDFromRequest returns the unique id associated to the request if any.
func IDFromRequest(r *http.Request, headerName string) (id string, ok bool) {
	if r == nil {
		return
	}
	id = r.Header.Get(headerName)
	if id != "" {
		return id, true
	}
	return IDFromCtx(r.Context())
}

// IDFromCtx returns the unique id associated to the context if any.
func IDFromCtx(ctx context.Context) (id string, ok bool) {
	id, ok = ctx.Value(cidKey{}).(string)
	return
}

// CtxWithID adds the given xid.ID to the context
func CtxWithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, cidKey{}, id)
}

func CorrelationIDHandler(fieldKey, headerName string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			id, ok := IDFromRequest(r, headerName)
			if !ok {
				id = xid.New().String()
				ctx = CtxWithID(ctx, id)
				r = r.WithContext(ctx)
			}
			if fieldKey != "" {
				log := zerolog.Ctx(ctx)
				log.UpdateContext(func(c zerolog.Context) zerolog.Context {
					return c.Str(fieldKey, id)
				})
			}
			if headerName != "" {
				w.Header().Set(headerName, id)
			}
			next.ServeHTTP(w, r)
		})
	}
}
