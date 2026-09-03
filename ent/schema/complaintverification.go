package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ComplaintVerification is the AI QA pass on top of complaint's keyword
// search (internal/domain/complaint): the keyword net over "دیر"+"برنامه"
// catches real delay complaints but also unrelated messages that happen to
// contain both words (e.g. "کورتیزول... تا دیروز... برنامه غذایی"). This
// stores GapGPT's verdict on whether one specific complaint excerpt is
// genuinely about late program delivery, keyed to the exact ComplaintAt it
// was verified against — if a customer's latest keyword-matched complaint
// changes (a newer one supersedes it), ComplaintAt no longer matches and
// the row is re-verified.
type ComplaintVerification struct {
	ent.Schema
}

func (ComplaintVerification) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("external_user_id", uuid.UUID{}).
			Immutable(),
		field.Time("complaint_at"),
		field.Bool("is_genuine"),
		field.Text("reasoning"),
		field.Time("verified_at"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (ComplaintVerification) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("external_user_id").Unique(),
	}
}
