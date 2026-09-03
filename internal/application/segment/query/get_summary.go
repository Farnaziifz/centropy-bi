package query

import (
	"context"
	"time"

	"centropy-affilate/internal/domain/segment"
	"centropy-affilate/internal/infrastructure/cache"
)

// summaryCacheTTL trades a little staleness for not re-running the
// segmentation query — a handful of joins over AlefGym's Users/Orders/
// Courses tables — on every admin dashboard load.
const summaryCacheTTL = 10 * time.Minute

const summaryCacheKey = "segment:summary"

type GetSummaryQuery struct{}

type GetSummaryHandler struct {
	repo  segment.Repository
	cache *cache.Cache
}

func NewGetSummaryHandler(repo segment.Repository, c *cache.Cache) *GetSummaryHandler {
	return &GetSummaryHandler{repo: repo, cache: c}
}

func (h *GetSummaryHandler) Handle(ctx context.Context, _ GetSummaryQuery) (segment.Summary, error) {
	if h.cache != nil {
		if cached, err := cache.Get[segment.Summary](ctx, h.cache, summaryCacheKey); err == nil {
			return cached, nil
		}
	}

	summary, err := h.repo.Summarize(ctx)
	if err != nil {
		return segment.Summary{}, err
	}

	if h.cache != nil {
		_ = cache.Set(ctx, h.cache, summaryCacheKey, summary, summaryCacheTTL)
	}
	return summary, nil
}
