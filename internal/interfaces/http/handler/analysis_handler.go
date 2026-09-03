package handler

import (
	"net/http"
	"strconv"

	analysiscmd "centropy-affilate/internal/application/analysis/command"
	analysisquery "centropy-affilate/internal/application/analysis/query"
	"centropy-affilate/internal/interfaces/http/dto"
	"centropy-affilate/pkg/cqrs"
)

// maxManualRunLimit bounds a manually-triggered run to a batch that still
// finishes within the LLM-batch route's timeout budget (4m30s — see
// router.go) — this handles synchronously within one HTTP request, so it
// must never be the unbounded daily job (see cmd/api/main.go's scheduler).
// 50 sequential GapGPT calls at a few seconds each comfortably clears that.
const maxManualRunLimit = 50

type AnalysisHandler struct {
	bus *cqrs.Bus
}

func NewAnalysisHandler(bus *cqrs.Bus) *AnalysisHandler {
	return &AnalysisHandler{bus: bus}
}

// ListOverdue returns the overdue-renewal list with whatever AI verdict
// exists for each customer so far.
//
//	@Summary		List overdue renewals with analysis
//	@Description	Overdue-renewal list with whatever AI verdict exists for each customer so far.
//	@Tags			analysis
//	@Produce		json
//	@Security		BearerAuth
//	@Param			days	query		int	false	"minimum days overdue (default 50)"
//	@Success		200		{array}		analysisquery.OverdueWithAnalysis
//	@Failure		401		{object}	dto.ErrorResponse
//	@Router			/admin/analysis/overdue [get]
func (h *AnalysisHandler) ListOverdue(w http.ResponseWriter, r *http.Request) {
	minDays := 50
	if raw := r.URL.Query().Get("days"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			minDays = parsed
		}
	}

	result, err := cqrs.ExecuteQuery[analysisquery.ListOverdueWithAnalysisQuery, []analysisquery.OverdueWithAnalysis](
		r.Context(), h.bus, analysisquery.ListOverdueWithAnalysisQuery{MinDays: minDays},
	)
	if err != nil {
		dto.WriteError(w, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, result)
}

// Run manually triggers a small, bounded analysis batch — for testing and
// for catching up a handful of customers without waiting for the next
// scheduled daily run.
//
//	@Summary		Trigger analysis run
//	@Description	Manually triggers a small, bounded analysis batch (default 5, max 50).
//	@Tags			analysis
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"batch size, capped at 50"
//	@Success		200		{object}	analysiscmd.RunDailyAnalysisResult
//	@Failure		401		{object}	dto.ErrorResponse
//	@Router			/admin/analysis/run [post]
func (h *AnalysisHandler) Run(w http.ResponseWriter, r *http.Request) {
	limit := 5
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > maxManualRunLimit {
		limit = maxManualRunLimit
	}

	result, err := cqrs.ExecuteCommand[analysiscmd.RunDailyAnalysisCommand, analysiscmd.RunDailyAnalysisResult](
		r.Context(), h.bus, analysiscmd.RunDailyAnalysisCommand{Limit: limit},
	)
	if err != nil {
		dto.WriteError(w, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, result)
}
