package httpmiddleware

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ErrorResponder func(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	code string,
	message string,
)

func RejectUnsafePaths(respond ErrorResponder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawPath := r.URL.EscapedPath()

			for _, rawSegment := range strings.Split(rawPath, "/") {
				segment, err := url.PathUnescape(rawSegment)
				if err != nil ||
					segment == "." ||
					segment == ".." ||
					strings.Contains(segment, "/") ||
					strings.Contains(segment, `\`) {
					respond(
						w,
						r,
						http.StatusBadRequest,
						"InvalidRequest",
						"invalid object key",
					)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func LogRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		// log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start))
		slog.Info("request handled",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", sw.status),
			slog.Float64("dur_ms", float64(time.Since(start))/float64(time.Millisecond)), // 0.626709
		)
	})
}
