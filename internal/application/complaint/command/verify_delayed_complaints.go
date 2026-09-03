package command

import (
	"context"
	"log/slog"
	"time"

	"centropy-affilate/internal/domain/complaint"
	apperrors "centropy-affilate/pkg/errors"
)

// VerifyDelayedComplaintsCommand runs GapGPT over the keyword-matched
// delayed-program complaint list to catch false positives (see
// complaint.Verification's doc) — one LLM call per complaint that's either
// never been verified or whose ComplaintAt has moved on since the last
// verification.
type VerifyDelayedComplaintsCommand struct {
	Limit int
}

type VerifyDelayedComplaintsResult struct {
	Candidates int `json:"candidates"`
	Verified   int `json:"verified"`
	Skipped    int `json:"skipped"`
	Failed     int `json:"failed"`
}

type VerifyDelayedComplaintsHandler struct {
	complaintRepo complaint.Repository
	verifyRepo    complaint.VerificationRepository
	verifier      complaint.Verifier
	log           *slog.Logger
}

func NewVerifyDelayedComplaintsHandler(
	complaintRepo complaint.Repository,
	verifyRepo complaint.VerificationRepository,
	verifier complaint.Verifier,
	log *slog.Logger,
) *VerifyDelayedComplaintsHandler {
	return &VerifyDelayedComplaintsHandler{
		complaintRepo: complaintRepo,
		verifyRepo:    verifyRepo,
		verifier:      verifier,
		log:           log,
	}
}

func (h *VerifyDelayedComplaintsHandler) Handle(ctx context.Context, cmd VerifyDelayedComplaintsCommand) (VerifyDelayedComplaintsResult, error) {
	candidates, err := h.complaintRepo.ListDelayedProgramComplainers(ctx)
	if err != nil {
		return VerifyDelayedComplaintsResult{}, apperrors.Wrap(apperrors.KindUnknown, "list delay complaint candidates", err)
	}

	result := VerifyDelayedComplaintsResult{Candidates: len(candidates)}

	for _, c := range candidates {
		if cmd.Limit > 0 && result.Verified >= cmd.Limit {
			break
		}

		existing, err := h.verifyRepo.FindByExternalUserID(ctx, c.ExternalUserID)
		if err == nil && existing.ComplaintAt.Equal(c.ComplaintAt) {
			// already verified this exact complaint — no new LLM call.
			result.Skipped++
			continue
		}
		if err != nil && apperrors.KindOf(err) != apperrors.KindNotFound {
			h.log.Error("verify: lookup existing failed", "user", c.ExternalUserID, "error", err)
			result.Failed++
			continue
		}

		isGenuine, reasoning, err := h.verifier.VerifyDelayComplaint(ctx, c.ComplaintExcerpt)
		if err != nil {
			h.log.Error("verify: classify failed", "user", c.ExternalUserID, "error", err)
			result.Failed++
			continue
		}

		if err := h.verifyRepo.Upsert(ctx, &complaint.Verification{
			ExternalUserID: c.ExternalUserID,
			ComplaintAt:    c.ComplaintAt,
			IsGenuine:      isGenuine,
			Reasoning:      reasoning,
			VerifiedAt:     time.Now().UTC(),
		}); err != nil {
			h.log.Error("verify: save failed", "user", c.ExternalUserID, "error", err)
			result.Failed++
			continue
		}
		result.Verified++
	}

	return result, nil
}
