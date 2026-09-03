package middleware

import (
	"log/slog"
	"net/http"

	"centropy-affilate/internal/interfaces/http/dto"
)

// Recover turns a panic anywhere downstream into a 500 JSON response
// instead of a dropped connection, and logs it with a stack trace.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered", "panic", rec, "path", r.URL.Path)
					dto.WriteJSON(w, http.StatusInternalServerError, dto.ErrorResponse{Error: "internal server error"})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
