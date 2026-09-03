// Package complaint mines AlefGym's support chat and tickets for customers
// who complained about a specific problem and never bought again — a
// heuristic, keyword-driven signal (there's no "reason for churn" field
// anywhere in AlefGym), not a certain causal claim. See
// DelayedProgramComplainer for the first instance of this: customers who
// complained about a late training/diet program and have placed zero
// completed orders since that complaint.
package complaint

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// DelayedProgramComplainer is a customer whose own chat/ticket message
// complained about program delivery being late, and who has completed zero
// orders since that message. "Since" is the operative word: a customer who
// complained and later bought again is excluded — the point of this list
// is customers this specific complaint may have actually cost.
type DelayedProgramComplainer struct {
	ExternalUserID        uuid.UUID
	Phone                 string
	FirstName             string
	LastName              string
	ComplaintAt           time.Time
	ComplaintExcerpt      string
	CompletedOrdersBefore int
}

// Verification is GapGPT's QA verdict on one DelayedProgramComplainer row:
// does the keyword-matched excerpt actually complain about a late program,
// or is it a false positive (the keyword net matches on word co-occurrence,
// not meaning — e.g. "کورتیزول... تا دیروز... برنامه غذایی" matches but
// isn't a delay complaint at all). Keyed to the exact ComplaintAt it was
// verified against, so a newer complaint superseding the old one forces
// re-verification rather than reusing a stale verdict.
type Verification struct {
	ExternalUserID uuid.UUID
	ComplaintAt    time.Time
	IsGenuine      bool
	Reasoning      string
	VerifiedAt     time.Time
}

// Verifier is the port to whatever LLM judges one complaint excerpt —
// internal/infrastructure/gapgpt implements this.
type Verifier interface {
	VerifyDelayComplaint(ctx context.Context, excerpt string) (isGenuine bool, reasoning string, err error)
}

// VerificationRepository is the local (own-DB) storage port for
// verification verdicts.
type VerificationRepository interface {
	Upsert(ctx context.Context, v *Verification) error
	FindByExternalUserID(ctx context.Context, externalUserID uuid.UUID) (*Verification, error)
	List(ctx context.Context) ([]Verification, error)
}
