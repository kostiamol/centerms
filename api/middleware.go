package api

import (
	"net/http"
	"time"
)

func (a *API) registerRoute(method, path string, handler http.HandlerFunc, middlewares ...func(next http.HandlerFunc, name string) http.HandlerFunc) {
	for _, mw := range middlewares {
		handler = mw(handler, path)
	}
	a.router.Handle(path, handler).Methods(method)
}

func (a *API) requestLogger(next http.HandlerFunc, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		a.log.With("method", r.Method, "uri", r.RequestURI, "name", name, "duration", time.Since(start)).Info()
	}
}
