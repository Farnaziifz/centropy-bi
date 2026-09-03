package query

import (
	"context"

	"github.com/google/uuid"

	"centropy-affilate/internal/domain/analysis"
	"centropy-affilate/internal/domain/renewal"
)

// OverdueWithAnalysis is one row for the admin table: the rule-based
// overdue fact (internal/domain/renewal) joined, in Go, with whatever AI
// verdict exists for that customer (internal/domain/analysis) — "pending"
// when the daily job hasn't reached them yet.
type OverdueWithAnalysis struct {
	renewal.OverdueCustomer
	Category   *analysis.Category `json:"Category,omitempty"`
	Summary    string             `json:"Summary,omitempty"`
	Confidence string             `json:"Confidence,omitempty"`
	AnalyzedAt *string            `json:"AnalyzedAt,omitempty"`
}

type ListOverdueWithAnalysisQuery struct {
	MinDays int
}

type ListOverdueWithAnalysisHandler struct {
	renewalRepo  renewal.Repository
	analysisRepo analysis.Repository
}

func NewListOverdueWithAnalysisHandler(renewalRepo renewal.Repository, analysisRepo analysis.Repository) *ListOverdueWithAnalysisHandler {
	return &ListOverdueWithAnalysisHandler{renewalRepo: renewalRepo, analysisRepo: analysisRepo}
}

func (h *ListOverdueWithAnalysisHandler) Handle(ctx context.Context, q ListOverdueWithAnalysisQuery) ([]OverdueWithAnalysis, error) {
	minDays := q.MinDays
	if minDays <= 0 {
		minDays = 50
	}

	overdue, err := h.renewalRepo.ListOverdueWithoutRepurchase(ctx, minDays)
	if err != nil {
		return nil, err
	}

	analyses, err := h.analysisRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	byUser := make(map[uuid.UUID]analysis.CustomerAnalysis, len(analyses))
	for _, a := range analyses {
		byUser[a.ExternalUserID] = a
	}

	out := make([]OverdueWithAnalysis, len(overdue))
	for i, c := range overdue {
		row := OverdueWithAnalysis{OverdueCustomer: c}
		if a, ok := byUser[c.ExternalUserID]; ok {
			cat := a.Category
			row.Category = &cat
			row.Summary = a.Summary
			row.Confidence = a.Confidence
			ts := a.AnalyzedAt.Format("2006-01-02T15:04:05Z07:00")
			row.AnalyzedAt = &ts
		}
		out[i] = row
	}
	return out, nil
}
