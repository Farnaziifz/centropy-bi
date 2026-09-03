// Package adminuser models the ops/growth-team login for this service's
// admin API.
package adminuser

import (
	"time"

	"github.com/google/uuid"
)

type AdminUser struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Name         string

	CreatedAt time.Time
	UpdatedAt time.Time
}

func New(email, passwordHash, name string) *AdminUser {
	now := time.Now().UTC()
	return &AdminUser{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
