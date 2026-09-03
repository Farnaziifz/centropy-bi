package handler

import (
	"net/http"
	"strconv"

	renewalquery "centropy-affilate/internal/application/renewal/query"
	"centropy-affilate/internal/domain/renewal"
	"centropy-affilate/internal/interfaces/http/dto"
	"centropy-affilate/pkg/cqrs"
)

type RenewalHandler struct {
	bus *cqrs.Bus
}

func NewRenewalHandler(bus *cqrs.Bus) *RenewalHandler {
	return &RenewalHandler{bus: bus}
}

// ListOverdue returns customers whose last delivered program is older than
// ?days= (default 50) with zero completed orders since.
func (h *RenewalHandler) ListOverdue(w http.ResponseWriter, r *http.Request) {
	minDays := 50
	if raw := r.URL.Query().Get("days"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			minDays = parsed
		}
	}

	result, err := cqrs.ExecuteQuery[renewalquery.ListOverdueQuery, []renewal.OverdueCustomer](
		r.Context(), h.bus, renewalquery.ListOverdueQuery{MinDays: minDays},
	)
	if err != nil {
		dto.WriteError(w, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, result)
}
