package analysis

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository is the local (own-DB) storage port for analysis results.
type Repository interface {
	Upsert(ctx context.Context, a *CustomerAnalysis) error
	FindByExternalUserID(ctx context.Context, externalUserID uuid.UUID) (*CustomerAnalysis, error)
	List(ctx context.Context) ([]CustomerAnalysis, error)
}

// CustomerMessage is one chat/ticket message authored by the customer
// themself, used as classifier input.
type CustomerMessage struct {
	Content   string
	CreatedAt time.Time
}

// MessageSource is the read-only port onto AlefGym's chat/ticket
// messages — internal/infrastructure/alefgym implements this. since nil
// means "from the beginning" (first-ever analysis of this customer).
type MessageSource interface {
	FetchCustomerMessages(ctx context.Context, externalUserID uuid.UUID, since *time.Time) ([]CustomerMessage, error)

	// LatestMessageAt returns, for every user in externalUserIDs who has at
	// least one message, the timestamp of their newest one — one query for
	// the whole batch. The daily/manual run uses this to skip "already
	// analyzed, nothing new" candidates without a per-candidate round trip
	// to AlefGym; only a user whose latest message is newer than their
	// stored cursor gets the (per-candidate) FetchCustomerMessages call.
	LatestMessageAt(ctx context.Context, externalUserIDs []uuid.UUID) (map[uuid.UUID]time.Time, error)
}
