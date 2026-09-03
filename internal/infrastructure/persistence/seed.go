package persistence

import (
	"context"
	"fmt"

	"centropy-affilate/ent"
	"centropy-affilate/internal/infrastructure/auth"
)

// SeedDefaultAdmin creates the first admin login from ADMIN_SEED_EMAIL /
// ADMIN_SEED_PASSWORD the first time the app boots against an empty
// AdminUser table — otherwise there's no way to get the first token to log
// in with. No-ops if either env var is unset, or if any admin already
// exists.
func SeedDefaultAdmin(ctx context.Context, client *ent.Client, email, password string) error {
	if email == "" || password == "" {
		return nil
	}

	count, err := client.AdminUser.Query().Count(ctx)
	if err != nil {
		return fmt.Errorf("count admin users: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash seed admin password: %w", err)
	}

	if _, err := client.AdminUser.Create().
		SetEmail(email).
		SetPasswordHash(hash).
		Save(ctx); err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}
	return nil
}
