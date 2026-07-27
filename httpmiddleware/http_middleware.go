package httpmiddleware

import (
	"net/http"
	"net/url"
	"strings"
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
