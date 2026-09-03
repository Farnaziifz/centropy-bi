package command

import (
	"context"
	"time"

	"centropy-affilate/internal/domain/customer"
	apperrors "centropy-affilate/pkg/errors"
)

type SyncCustomersCommand struct{}

type SyncCustomersResult struct {
	Synced int `json:"synced"`
}

// SyncCustomersHandler pulls the full AlefGym user directory (via
// customer.Source) and upserts it into this service's local Customer
// table. Triggered manually from the admin API for now (POST
// /admin/customers/sync) — a scheduled job is the natural next step once
// this is in daily use, but isn't needed to make the loyalty-club
// dashboard (loyalty-club-roadmap.html phase 1) usable today.
type SyncCustomersHandler struct {
	source customer.Source
	repo   customer.Repository
}

func NewSyncCustomersHandler(source customer.Source, repo customer.Repository) *SyncCustomersHandler {
	return &SyncCustomersHandler{source: source, repo: repo}
}

func (h *SyncCustomersHandler) Handle(ctx context.Context, _ SyncCustomersCommand) (SyncCustomersResult, error) {
	rows, err := h.source.FetchAll(ctx)
	if err != nil {
		return SyncCustomersResult{}, err
	}

	now := time.Now().UTC()
	for i := range rows {
		rows[i].LastSyncedAt = now
		if err := h.repo.Upsert(ctx, &rows[i]); err != nil {
			return SyncCustomersResult{}, apperrors.Wrap(apperrors.KindUnknown, "sync customers", err)
		}
	}

	return SyncCustomersResult{Synced: len(rows)}, nil
}
