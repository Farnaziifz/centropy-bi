package persistence

import (
	"context"

	"centropy-affilate/ent"
	entadminuser "centropy-affilate/ent/adminuser"
	"centropy-affilate/internal/domain/adminuser"
	apperrors "centropy-affilate/pkg/errors"
)

type AdminUserRepository struct {
	client *ent.Client
}

func NewAdminUserRepository(client *ent.Client) *AdminUserRepository {
	return &AdminUserRepository{client: client}
}

func (r *AdminUserRepository) Create(ctx context.Context, u *adminuser.AdminUser) error {
	_, err := r.client.AdminUser.Create().
		SetID(u.ID).
		SetEmail(u.Email).
		SetPasswordHash(u.PasswordHash).
		SetName(u.Name).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return apperrors.Conflict("an admin with this email already exists")
		}
		return apperrors.Wrap(apperrors.KindUnknown, "create admin user", err)
	}
	return nil
}

func (r *AdminUserRepository) FindByEmail(ctx context.Context, email string) (*adminuser.AdminUser, error) {
	row, err := r.client.AdminUser.Query().
		Where(entadminuser.EmailEQ(email)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, apperrors.Unauthorized("invalid email or password")
		}
		return nil, apperrors.Wrap(apperrors.KindUnknown, "find admin user by email", err)
	}
	return toDomainAdminUser(row), nil
}

func toDomainAdminUser(row *ent.AdminUser) *adminuser.AdminUser {
	return &adminuser.AdminUser{
		ID:           row.ID,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Name:         row.Name,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
