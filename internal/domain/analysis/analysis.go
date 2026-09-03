// Package analysis is the AI layer on top of internal/domain/renewal: for
// each customer overdue for a repurchase, read their own chat/ticket
// messages with GapGPT and classify why they likely stopped buying —
// sharper than complaint's keyword search, at the cost of an LLM call per
// customer. Results are stored locally (own DB) since a call costs money
// and takes seconds; the daily job (see application/analysis/command)
// only re-analyzes a customer once new messages exist past their stored
// cursor.
package analysis

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Category string

const (
	ProgramDelay   Category = "PROGRAM_DELAY"   // complained their program/diet was late or never arrived
	Price          Category = "PRICE"           // concerned about cost, asked for discount/installments
	SupportQuality Category = "SUPPORT_QUALITY" // complained support/coach was slow or unhelpful
	TechnicalIssue Category = "TECHNICAL_ISSUE" // app bugs, upload/playback errors, payment gateway errors
	HealthPersonal Category = "HEALTH_PERSONAL" // injury, illness, or personal-life reason unrelated to the product
	LostInterest   Category = "LOST_INTEREST"   // no complaint at all, reads as simply gone quiet
	Unclear        Category = "UNCLEAR"         // messages exist but give no clear signal either way
	NoMessages     Category = "NO_MESSAGES"     // customer has never sent a single chat/ticket message
)

// CustomerAnalysis is one customer's stored AI verdict.
type CustomerAnalysis struct {
	ExternalUserID uuid.UUID
	Category       Category
	Summary        string
	Confidence     string // "high" | "medium" | "low"
	// LastMessageAt is nil only for NO_MESSAGES; otherwise it's the
	// timestamp of the newest message considered in this analysis — the
	// daily job's re-analysis cursor.
	LastMessageAt *time.Time
	AnalyzedAt    time.Time
}

// Classifier is the port to whatever LLM reads a customer's messages and
// produces a verdict — internal/infrastructure/gapgpt implements this.
// messages is the customer's own messages, chronological.
type Classifier interface {
	Classify(ctx context.Context, messages []string) (category Category, summary string, confidence string, err error)
}
