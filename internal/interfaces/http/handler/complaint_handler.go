package handler

import (
	"net/http"
	"strconv"

	complaintcmd "centropy-affilate/internal/application/complaint/command"
	complaintquery "centropy-affilate/internal/application/complaint/query"
	"centropy-affilate/internal/domain/complaint"
	"centropy-affilate/internal/interfaces/http/dto"
	"centropy-affilate/pkg/cqrs"
)

// maxManualVerifyLimit bounds a manually-triggered verification run — see
// AnalysisHandler.Run's identical reasoning.
const maxManualVerifyLimit = 20

type ComplaintHandler struct {
	bus *cqrs.Bus
}

func NewComplaintHandler(bus *cqrs.Bus) *ComplaintHandler {
	return &ComplaintHandler{bus: bus}
}

// ListDelayedProgramComplainers returns customers who complained about a
// late program and have bought nothing since.
func (h *ComplaintHandler) ListDelayedProgramComplainers(w http.ResponseWriter, r *http.Request) {
	result, err := cqrs.ExecuteQuery[complaintquery.ListDelayedProgramComplainersQuery, []complaint.DelayedProgramComplainer](
		r.Context(), h.bus, complaintquery.ListDelayedProgramComplainersQuery{},
	)
	if err != nil {
		dto.WriteError(w, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, result)
}

// ListVerified returns the delayed-program complaint list with GapGPT's
// genuine/false-positive verdict, where available.
func (h *ComplaintHandler) ListVerified(w http.ResponseWriter, r *http.Request) {
	result, err := cqrs.ExecuteQuery[complaintquery.ListVerifiedComplainersQuery, []complaintquery.VerifiedComplainer](
		r.Context(), h.bus, complaintquery.ListVerifiedComplainersQuery{},
	)
	if err != nil {
		dto.WriteError(w, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, result)
}

// Verify manually triggers a small, bounded verification batch.
func (h *ComplaintHandler) Verify(w http.ResponseWriter, r *http.Request) {
	limit := 5
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > maxManualVerifyLimit {
		limit = maxManualVerifyLimit
	}

	result, err := cqrs.ExecuteCommand[complaintcmd.VerifyDelayedComplaintsCommand, complaintcmd.VerifyDelayedComplaintsResult](
		r.Context(), h.bus, complaintcmd.VerifyDelayedComplaintsCommand{Limit: limit},
	)
	if err != nil {
		dto.WriteError(w, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, result)
}
