package app

import (
	"net/http"
	"time"
)

// responseRecorder is used to record what status is returned, so we can log it
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader is a function from the http.ResponseWriter interface
func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// LogRoute returns a handler that logs the requests
func LogRoute(a *App, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default to 200 OK
		}
		a.Logger.Debug("incoming request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
		a.Logger.Debug("completed request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start), "statuscode", recorder.statusCode)
	})
}
