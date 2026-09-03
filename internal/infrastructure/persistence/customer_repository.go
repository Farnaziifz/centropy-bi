package persistence

import (
	"context"

	"github.com/google/uuid"

	"centropy-affilate/ent"
	entcustomer "centropy-affilate/ent/customer"
	"centropy-affilate/internal/domain/customer"
	apperrors "centropy-affilate/pkg/errors"
)

type CustomerRepository struct {
	client *ent.Client
}

func NewCustomerRepository(client *ent.Client) *CustomerRepository {
	return &CustomerRepository{client: client}
}

// Upsert relies on ent's generated ON CONFLICT clause (sql/upsert feature,
// enabled in ent/generate.go) keyed on the unique external_user_id index,
// so a repeated sync run updates the existing row instead of erroring. The
// row's local ID is never set explicitly: a fresh one is assigned by the
// schema default on first insert, and left untouched on every later
// conflict/update.
func (r *CustomerRepository) Upsert(ctx context.Context, c *customer.Customer) error {
	err := r.client.Customer.Create().
		SetExternalUserID(c.ExternalUserID).
		SetPhone(c.Phone).
		SetFirstName(c.FirstName).
		SetLastName(c.LastName).
		SetRegisteredAt(c.RegisteredAt).
		SetLastSyncedAt(c.LastSyncedAt).
		OnConflictColumns(entcustomer.FieldExternalUserID).
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return apperrors.Wrap(apperrors.KindUnknown, "upsert customer", err)
	}
	return nil
}

func (r *CustomerRepository) FindByExternalUserID(ctx context.Context, externalUserID uuid.UUID) (*customer.Customer, error) {
	row, err := r.client.Customer.Query().
		Where(entcustomer.ExternalUserIDEQ(externalUserID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, apperrors.NotFound("customer not found")
		}
		return nil, apperrors.Wrap(apperrors.KindUnknown, "find customer by external user id", err)
	}
	return toDomainCustomer(row), nil
}

func (r *CustomerRepository) List(ctx context.Context) ([]customer.Customer, error) {
	rows, err := r.client.Customer.Query().
		Order(ent.Asc(entcustomer.FieldFirstName)).
		All(ctx)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "list customers", err)
	}
	out := make([]customer.Customer, len(rows))
	for i, row := range rows {
		out[i] = *toDomainCustomer(row)
	}
	return out, nil
}

func toDomainCustomer(row *ent.Customer) *customer.Customer {
	return &customer.Customer{
		ID:             row.ID,
		ExternalUserID: row.ExternalUserID,
		Phone:          row.Phone,
		FirstName:      row.FirstName,
		LastName:       row.LastName,
		RegisteredAt:   row.RegisteredAt,
		LastSyncedAt:   row.LastSyncedAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}
