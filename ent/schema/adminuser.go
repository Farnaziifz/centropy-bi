package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// AdminUser is an ops/growth-team login for this service's own admin API —
// separate from AlefGym's customer accounts, since this backend is a
// standalone analytics+loyalty service with its own database (see
// internal/infrastructure/alefgym for the read-only link back to the
// AlefGym source of truth).
type AdminUser struct {
	ent.Schema
}

func (AdminUser) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("email").
			NotEmpty().
			Unique().
			MaxLen(160),
		field.String("password_hash").
			NotEmpty().
			Sensitive(),
		field.String("name").
			Optional().
			MaxLen(120),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}
