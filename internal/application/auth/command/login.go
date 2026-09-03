package command

import (
	"context"
	"time"

	"github.com/google/uuid"

	"centropy-affilate/internal/domain/adminuser"
	apperrors "centropy-affilate/pkg/errors"
)

// TokenIssuer is satisfied structurally by infrastructure/auth.JWTService —
// defined here, on the consumer side, so the application layer never
// imports infrastructure directly.
type TokenIssuer interface {
	Issue(adminUserID uuid.UUID) (string, time.Time, error)
}

// PasswordVerifier is satisfied structurally by
// infrastructure/auth.VerifyPassword.
type PasswordVerifier func(hash, plain string) bool

type LoginCommand struct {
	Email    string
	Password string
}

type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type LoginHandler struct {
	repo   adminuser.Repository
	tokens TokenIssuer
	verify PasswordVerifier
}

func NewLoginHandler(repo adminuser.Repository, tokens TokenIssuer, verify PasswordVerifier) *LoginHandler {
	return &LoginHandler{repo: repo, tokens: tokens, verify: verify}
}

func (h *LoginHandler) Handle(ctx context.Context, cmd LoginCommand) (LoginResult, error) {
	u, err := h.repo.FindByEmail(ctx, cmd.Email)
	if err != nil {
		return LoginResult{}, err
	}

	if !h.verify(u.PasswordHash, cmd.Password) {
		return LoginResult{}, apperrors.Unauthorized("invalid email or password")
	}

	token, expiresAt, err := h.tokens.Issue(u.ID)
	if err != nil {
		return LoginResult{}, apperrors.Wrap(apperrors.KindUnknown, "issue token", err)
	}

	return LoginResult{Token: token, ExpiresAt: expiresAt}, nil
}
