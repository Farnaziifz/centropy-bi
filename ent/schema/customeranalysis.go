package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// CustomerAnalysis is the AI-derived verdict on why one customer (from the
// renewal.OverdueCustomer population — see internal/domain/renewal) hasn't
// bought again, read straight from their own chat/ticket messages by
// GapGPT. LastMessageAt is the cursor the daily job re-analyzes from: a
// customer is only re-sent to the LLM once messages exist after this
// timestamp, so "read new chats daily" costs nothing for someone who's
// gone quiet.
type CustomerAnalysis struct {
	ent.Schema
}

func (CustomerAnalysis) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("external_user_id", uuid.UUID{}).
			Immutable(),
		field.Enum("category").
			Values(
				"PROGRAM_DELAY", "PRICE", "SUPPORT_QUALITY", "TECHNICAL_ISSUE",
				"HEALTH_PERSONAL", "LOST_INTEREST", "UNCLEAR", "NO_MESSAGES",
			),
		field.Text("summary"),
		field.Enum("confidence").
			Values("high", "medium", "low").
			Default("low"),
		field.Time("last_message_at").
			Optional().
			Nillable(),
		field.Time("analyzed_at"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (CustomerAnalysis) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("external_user_id").Unique(),
	}
}
