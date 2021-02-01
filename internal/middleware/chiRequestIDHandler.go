package middleware

import (
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
)

// ChiRequestIDHandler returns a handler attaching the unique id bound to the request to zerlogo.
// In the Chi ecosystem this id gets attached by the github.com/go-chi/chi/middleware.RequestID middleware.
// This handler also takes care of adding the id to the response unless no headerName is provided.
// fieldKey describes the field in zerologs output.
//
// This really is just an adapter making zerolog aware of this already set ID
// so one does not have to attach another one not playing well in the context of Chi.
//
// It mimics the RequestIDHandler contained in the zerolog library:
// https://github.com/rs/zerolog/blob/a8f5328bb7c784b044cc9649643d56d97ad2334c/hlog/hlog.go#L150
func ChiRequestIDHandler(fieldKey, headerName string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			id := middleware.GetReqID(ctx)

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
