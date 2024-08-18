package middlewares

import (
	"go.uber.org/zap"
	"net/http"
)

func LoggerMW(logger zap.SugaredLogger) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Infof("serving a request: method %v, url %v", r.Method, r.URL)
			next.ServeHTTP(w, r)
		})
	}
}
