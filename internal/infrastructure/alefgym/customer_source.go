package alefgym

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"centropy-affilate/internal/domain/customer"
	apperrors "centropy-affilate/pkg/errors"
)

// CustomerSource implements customer.Source by reading every non-deleted
// AlefGym user. At AlefGym's current scale a full pull is simpler and
// cheap enough that an incremental/delta sync isn't worth the extra state
// (see customer.Source's doc comment).
type CustomerSource struct {
	db              *sql.DB
	excludedUserIDs []uuid.UUID
}

func NewCustomerSource(db *sql.DB, excludedUserIDs []uuid.UUID) *CustomerSource {
	return &CustomerSource{db: db, excludedUserIDs: excludedUserIDs}
}

func (s *CustomerSource) excludedAsText() []string {
	out := make([]string, len(s.excludedUserIDs))
	for i, id := range s.excludedUserIDs {
		out[i] = id.String()
	}
	return out
}

func (s *CustomerSource) FetchAll(ctx context.Context) ([]customer.Customer, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT "ID", "PhoneNumber", "FirstName", "LastName", "CreatedAt"
		FROM "AUTHENTICATION"."Users"
		WHERE "IsDeleted" = false
		  AND "ID"::text <> ALL($1::text[])
	`, s.excludedAsText())
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "fetch alefgym customers", err)
	}
	defer rows.Close()

	var out []customer.Customer
	for rows.Next() {
		var c customer.Customer
		var firstName, lastName sql.NullString
		if err := rows.Scan(&c.ExternalUserID, &c.Phone, &firstName, &lastName, &c.RegisteredAt); err != nil {
			return nil, apperrors.Wrap(apperrors.KindUnknown, "scan alefgym customer", err)
		}
		c.FirstName = firstName.String
		c.LastName = lastName.String
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "iterate alefgym customers", err)
	}
	return out, nil
}
