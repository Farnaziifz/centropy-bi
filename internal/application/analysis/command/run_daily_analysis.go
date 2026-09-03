package command

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"centropy-affilate/internal/domain/analysis"
	"centropy-affilate/internal/domain/renewal"
	apperrors "centropy-affilate/pkg/errors"
)

// RunDailyAnalysisCommand re-analyzes the renewal.OverdueCustomer
// population (>=50 days since last program, no repurchase since — see
// internal/domain/renewal): any customer never analyzed, or with chat/
// ticket messages newer than their stored cursor, gets one GapGPT call.
// Everyone else costs nothing to skip. Limit caps how many customers get
// an LLM call in this invocation — the manual "run now" trigger passes a
// small number; the unattended daily job passes 0 (no cap).
type RunDailyAnalysisCommand struct {
	Limit int
}

type RunDailyAnalysisResult struct {
	Candidates int `json:"candidates"`
	Analyzed   int `json:"analyzed"`
	Skipped    int `json:"skipped"`
	Failed     int `json:"failed"`
}

type RunDailyAnalysisHandler struct {
	renewalRepo   renewal.Repository
	analysisRepo  analysis.Repository
	messageSource analysis.MessageSource
	classifier    analysis.Classifier
	log           *slog.Logger
}

func NewRunDailyAnalysisHandler(
	renewalRepo renewal.Repository,
	analysisRepo analysis.Repository,
	messageSource analysis.MessageSource,
	classifier analysis.Classifier,
	log *slog.Logger,
) *RunDailyAnalysisHandler {
	return &RunDailyAnalysisHandler{
		renewalRepo:   renewalRepo,
		analysisRepo:  analysisRepo,
		messageSource: messageSource,
		classifier:    classifier,
		log:           log,
	}
}

func (h *RunDailyAnalysisHandler) Handle(ctx context.Context, cmd RunDailyAnalysisCommand) (RunDailyAnalysisResult, error) {
	candidates, err := h.renewalRepo.ListOverdueWithoutRepurchase(ctx, 50)
	if err != nil {
		return RunDailyAnalysisResult{}, apperrors.Wrap(apperrors.KindUnknown, "list overdue candidates", err)
	}

	result := RunDailyAnalysisResult{Candidates: len(candidates)}

	// Two batch lookups instead of two AlefGym/DB round trips PER
	// candidate — with 400+ candidates and most of them unchanged run to
	// run, per-candidate lookups made every batch slower than the last
	// (rescanning everyone already done) and eventually blew the request
	// timeout. See analysis.MessageSource.LatestMessageAt's doc.
	existingList, err := h.analysisRepo.List(ctx)
	if err != nil {
		return RunDailyAnalysisResult{}, apperrors.Wrap(apperrors.KindUnknown, "list existing analyses", err)
	}
	existingByUser := make(map[uuid.UUID]analysis.CustomerAnalysis, len(existingList))
	for _, a := range existingList {
		existingByUser[a.ExternalUserID] = a
	}

	candidateIDs := make([]uuid.UUID, len(candidates))
	for i, c := range candidates {
		candidateIDs[i] = c.ExternalUserID
	}
	latestByUser, err := h.messageSource.LatestMessageAt(ctx, candidateIDs)
	if err != nil {
		return RunDailyAnalysisResult{}, apperrors.Wrap(apperrors.KindUnknown, "fetch latest message timestamps", err)
	}

	for _, c := range candidates {
		if cmd.Limit > 0 && result.Analyzed >= cmd.Limit {
			break
		}
		if ctx.Err() != nil {
			// caller (HTTP request context, most often) went away or hit
			// its deadline — stop cleanly and return what's been done so
			// far, instead of spending the rest of the candidate list
			// racking up "context canceled" on every remaining call.
			h.log.Warn("analysis: stopping early, context done", "error", ctx.Err(), "processed_of", len(candidates))
			break
		}

		existing, hasExisting := existingByUser[c.ExternalUserID]
		latestMsg, hasMsg := latestByUser[c.ExternalUserID]

		if !hasMsg {
			if !hasExisting {
				// never had any messages, ever — no LLM call, just record
				// it once so the admin list doesn't show it as blank.
				if err := h.analysisRepo.Upsert(ctx, &analysis.CustomerAnalysis{
					ExternalUserID: c.ExternalUserID,
					Category:       analysis.NoMessages,
					Summary:        "این کاربر هیچ پیام چت یا تیکتی ثبت نکرده است.",
					Confidence:     "high",
					AnalyzedAt:     time.Now().UTC(),
				}); err != nil {
					h.log.Error("analysis: save NO_MESSAGES failed", "user", c.ExternalUserID, "error", err)
					result.Failed++
					continue
				}
			}
			result.Skipped++
			continue
		}

		if hasExisting && existing.LastMessageAt != nil && !latestMsg.After(*existing.LastMessageAt) {
			// already analyzed, nothing new since — the whole point of the
			// cursor: zero network calls for this candidate.
			result.Skipped++
			continue
		}

		var since *time.Time
		if hasExisting {
			since = existing.LastMessageAt
		}
		messages, err := h.messageSource.FetchCustomerMessages(ctx, c.ExternalUserID, since)
		if err != nil {
			h.log.Error("analysis: fetch messages failed", "user", c.ExternalUserID, "error", err)
			result.Failed++
			continue
		}
		if len(messages) == 0 {
			// LatestMessageAt said there was something, but it's already
			// covered by the stored cursor by the time we actually fetched
			// (a race with the cheap check above) — nothing to do.
			result.Skipped++
			continue
		}

		texts := make([]string, len(messages))
		latest := messages[0].CreatedAt
		for i, m := range messages {
			texts[i] = m.Content
			if m.CreatedAt.After(latest) {
				latest = m.CreatedAt
			}
		}

		category, summary, confidence, err := h.classifier.Classify(ctx, texts)
		if err != nil {
			h.log.Error("analysis: classify failed", "user", c.ExternalUserID, "error", err)
			result.Failed++
			continue
		}

		if err := h.analysisRepo.Upsert(ctx, &analysis.CustomerAnalysis{
			ExternalUserID: c.ExternalUserID,
			Category:       category,
			Summary:        summary,
			Confidence:     confidence,
			LastMessageAt:  &latest,
			AnalyzedAt:     time.Now().UTC(),
		}); err != nil {
			h.log.Error("analysis: save failed", "user", c.ExternalUserID, "error", err)
			result.Failed++
			continue
		}
		result.Analyzed++
	}

	return result, nil
}
