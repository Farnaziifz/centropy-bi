// Package customer models the local, synced-from-AlefGym customer
// directory. See ent/schema/customer.go for why this table exists
// alongside AlefGym remaining the source of truth.
package customer

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID             uuid.UUID
	ExternalUserID uuid.UUID
	Phone          string
	FirstName      string
	LastName       string
	RegisteredAt   time.Time
	LastSyncedAt   time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
