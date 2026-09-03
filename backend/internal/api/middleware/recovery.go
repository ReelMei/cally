package middleware

import (
	"fmt"
	"log/slog"
	"net/http"

	"cally/pkg/response"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered in HTTP handler",
					"error", fmt.Sprintf("%v", err),
					"path", r.URL.Path,
				)
				response.Error(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected server error occurred")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
