package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	segmentquery "centropy-affilate/internal/application/segment/query"
	"centropy-affilate/internal/domain/segment"
	"centropy-affilate/internal/interfaces/http/dto"
	"centropy-affilate/pkg/cqrs"
)

type SegmentHandler struct {
	bus *cqrs.Bus
}

func NewSegmentHandler(bus *cqrs.Bus) *SegmentHandler {
	return &SegmentHandler{bus: bus}
}

// Summary returns the six-segment breakdown from loyalty-club-roadmap.html
// — newcomer / cold / hero / at-risk / churned / one-time — plus the total
// customer count it was computed over.
func (h *SegmentHandler) Summary(w http.ResponseWriter, r *http.Request) {
	result, err := cqrs.ExecuteQuery[segmentquery.GetSummaryQuery, segment.Summary](
		r.Context(), h.bus, segmentquery.GetSummaryQuery{},
	)
	if err != nil {
		dto.WriteError(w, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, result)
}

// ListNonPurchasers returns every registered customer who has never
// completed a single order (Newcomer + Cold combined, no 14-day split).
func (h *SegmentHandler) ListNonPurchasers(w http.ResponseWriter, r *http.Request) {
	result, err := cqrs.ExecuteQuery[segmentquery.ListNonPurchasersQuery, []segment.NonPurchaser](
		r.Context(), h.bus, segmentquery.ListNonPurchasersQuery{},
	)
	if err != nil {
		dto.WriteError(w, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, result)
}

// MonthlyNonPurchaserSignups returns, per signup month, how many of that
// month's registrants still have zero completed orders today — the chart
// behind "هر ماه چند ثبت‌نامی داشتیم که هیچ خریدی انجام نداده‌اند".
func (h *SegmentHandler) MonthlyNonPurchaserSignups(w http.ResponseWriter, r *http.Request) {
	result, err := cqrs.ExecuteQuery[segmentquery.MonthlyNonPurchaserSignupsQuery, []segment.MonthlySignups](
		r.Context(), h.bus, segmentquery.MonthlyNonPurchaserSignupsQuery{},
	)
	if err != nil {
		dto.WriteError(w, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, result)
}

// ListCustomers returns every customer currently in one segment.
func (h *SegmentHandler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	seg := segment.Segment(chi.URLParam(r, "segment"))

	result, err := cqrs.ExecuteQuery[segmentquery.ListCustomersQuery, []segment.Customer](
		r.Context(), h.bus, segmentquery.ListCustomersQuery{Segment: seg},
	)
	if err != nil {
		dto.WriteError(w, err)
		return
	}
	dto.WriteJSON(w, http.StatusOK, result)
}
