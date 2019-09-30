package api

import (
	"net/http"
	"time"

	"github.com/kostiamol/centerms/log"
)

func (a *api) registerRoute(method, path string, handler http.HandlerFunc, middlewares ...func(next http.HandlerFunc, name string, l log.Logger) http.HandlerFunc) {
	for _, mw := range middlewares {
		handler = mw(handler, path, a.log)
	}
	a.router.Handle(path, handler).Methods(method)
}

func requestLogger(next http.HandlerFunc, name string, l log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		l.With("method", r.Method, "uri", r.RequestURI, "name", name, "duration", time.Since(start)).Info()
	}
}
