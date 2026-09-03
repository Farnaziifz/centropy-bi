package customer

import (
	"context"

	"github.com/google/uuid"
)

// Source is the read-only port onto AlefGym — the actual implementation
// (internal/infrastructure/alefgym) runs plain SQL against the AlefGym
// production database. It returns the whole directory on every sync call;
// at AlefGym's current scale (~1k users) a full pull is simpler and cheap
// enough that an incremental/delta sync isn't worth the extra state.
type Source interface {
	FetchAll(ctx context.Context) ([]Customer, error)
}

// Repository is the local persistence port for the synced directory.
type Repository interface {
	// Upsert inserts or updates a customer by ExternalUserID.
	Upsert(ctx context.Context, c *Customer) error
	FindByExternalUserID(ctx context.Context, externalUserID uuid.UUID) (*Customer, error)
	List(ctx context.Context) ([]Customer, error)
}
