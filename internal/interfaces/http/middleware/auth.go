package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"centropy-affilate/internal/interfaces/http/dto"
)

type ctxKey int

const adminUserIDKey ctxKey = iota

// TokenParser is satisfied structurally by infrastructure/auth.JWTService.
type TokenParser interface {
	Parse(raw string) (uuid.UUID, error)
}

// RequireAuth extracts and validates a "Bearer <jwt>" token, stashing the
// admin user id in the request context for handlers to read via
// AdminUserID(ctx). Every route in this API is an internal admin route —
// there's no customer-facing auth here, that stays in AlefGym.
func RequireAuth(parser TokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || token == "" {
				dto.WriteJSON(w, http.StatusUnauthorized, dto.ErrorResponse{Error: "missing bearer token"})
				return
			}

			adminUserID, err := parser.Parse(token)
			if err != nil {
				dto.WriteJSON(w, http.StatusUnauthorized, dto.ErrorResponse{Error: "invalid or expired token"})
				return
			}

			ctx := context.WithValue(r.Context(), adminUserIDKey, adminUserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminUserID reads the authenticated admin user id set by RequireAuth.
// Only call this from handlers mounted behind RequireAuth.
func AdminUserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(adminUserIDKey).(uuid.UUID)
	return id, ok
}
