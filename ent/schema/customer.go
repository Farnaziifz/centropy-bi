package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Customer is a local, synced-from-AlefGym directory row. AlefGym stays the
// source of truth for identity and purchase history (see
// internal/infrastructure/alefgym) — this table exists so admin screens can
// search/browse customers without hitting the remote production database on
// every request, and so future loyalty features (points, referrals) have a
// local row to hang state off of. ExternalUserID is AlefGym's
// AUTHENTICATION.Users.ID.
type Customer struct {
	ent.Schema
}

func (Customer) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("external_user_id", uuid.UUID{}).
			Immutable(),
		field.String("phone").
			NotEmpty().
			MaxLen(20),
		field.String("first_name").
			Optional().
			MaxLen(200),
		field.String("last_name").
			Optional().
			MaxLen(200),
		field.Time("registered_at"),
		field.Time("last_synced_at"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Customer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("external_user_id").Unique(),
		index.Fields("phone"),
	}
}
