package query

import (
	"context"
	"time"

	"centropy-affilate/internal/domain/complaint"
	"centropy-affilate/internal/infrastructure/cache"
)

// cacheTTL is longer than the segment queries' 10 minutes — this one scans
// full message/ticket text rather than indexed columns, so it's the
// heaviest query in this service.
const cacheTTL = 30 * time.Minute
const cacheKey = "complaint:delayed-program"

type ListDelayedProgramComplainersQuery struct{}

type ListDelayedProgramComplainersHandler struct {
	repo  complaint.Repository
	cache *cache.Cache
}

func NewListDelayedProgramComplainersHandler(repo complaint.Repository, c *cache.Cache) *ListDelayedProgramComplainersHandler {
	return &ListDelayedProgramComplainersHandler{repo: repo, cache: c}
}

func (h *ListDelayedProgramComplainersHandler) Handle(ctx context.Context, _ ListDelayedProgramComplainersQuery) ([]complaint.DelayedProgramComplainer, error) {
	if h.cache != nil {
		if cached, err := cache.Get[[]complaint.DelayedProgramComplainer](ctx, h.cache, cacheKey); err == nil {
			return cached, nil
		}
	}

	rows, err := h.repo.ListDelayedProgramComplainers(ctx)
	if err != nil {
		return nil, err
	}

	if h.cache != nil {
		_ = cache.Set(ctx, h.cache, cacheKey, rows, cacheTTL)
	}
	return rows, nil
}
