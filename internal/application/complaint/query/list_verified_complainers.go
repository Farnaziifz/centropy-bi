package query

import (
	"context"

	"github.com/google/uuid"

	"centropy-affilate/internal/domain/complaint"
)

// VerifiedComplainer is one row for the admin table: the keyword-matched
// fact joined, in Go, with GapGPT's verdict on whether it's genuine —
// Verified is nil when the daily verification job hasn't reached this
// complaint yet.
type VerifiedComplainer struct {
	complaint.DelayedProgramComplainer
	IsGenuine *bool  `json:"IsGenuine,omitempty"`
	Reasoning string `json:"Reasoning,omitempty"`
}

type ListVerifiedComplainersQuery struct{}

type ListVerifiedComplainersHandler struct {
	complaintRepo complaint.Repository
	verifyRepo    complaint.VerificationRepository
}

func NewListVerifiedComplainersHandler(complaintRepo complaint.Repository, verifyRepo complaint.VerificationRepository) *ListVerifiedComplainersHandler {
	return &ListVerifiedComplainersHandler{complaintRepo: complaintRepo, verifyRepo: verifyRepo}
}

func (h *ListVerifiedComplainersHandler) Handle(ctx context.Context, _ ListVerifiedComplainersQuery) ([]VerifiedComplainer, error) {
	candidates, err := h.complaintRepo.ListDelayedProgramComplainers(ctx)
	if err != nil {
		return nil, err
	}

	verifications, err := h.verifyRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	byUser := make(map[uuid.UUID]complaint.Verification, len(verifications))
	for _, v := range verifications {
		byUser[v.ExternalUserID] = v
	}

	out := make([]VerifiedComplainer, len(candidates))
	for i, c := range candidates {
		row := VerifiedComplainer{DelayedProgramComplainer: c}
		if v, ok := byUser[c.ExternalUserID]; ok && v.ComplaintAt.Equal(c.ComplaintAt) {
			genuine := v.IsGenuine
			row.IsGenuine = &genuine
			row.Reasoning = v.Reasoning
		}
		out[i] = row
	}
	return out, nil
}
