package query

import (
	"context"
	"fmt"
	"time"

	"centropy-affilate/internal/domain/renewal"
	"centropy-affilate/internal/infrastructure/cache"
)

const cacheTTL = 10 * time.Minute

type ListOverdueQuery struct {
	MinDays int
}

type ListOverdueHandler struct {
	repo  renewal.Repository
	cache *cache.Cache
}

func NewListOverdueHandler(repo renewal.Repository, c *cache.Cache) *ListOverdueHandler {
	return &ListOverdueHandler{repo: repo, cache: c}
}

func (h *ListOverdueHandler) Handle(ctx context.Context, q ListOverdueQuery) ([]renewal.OverdueCustomer, error) {
	minDays := q.MinDays
	if minDays <= 0 {
		minDays = 50
	}
	cacheKey := fmt.Sprintf("renewal:overdue:%d", minDays)

	if h.cache != nil {
		if cached, err := cache.Get[[]renewal.OverdueCustomer](ctx, h.cache, cacheKey); err == nil {
			return cached, nil
		}
	}

	rows, err := h.repo.ListOverdueWithoutRepurchase(ctx, minDays)
	if err != nil {
		return nil, err
	}

	if h.cache != nil {
		_ = cache.Set(ctx, h.cache, cacheKey, rows, cacheTTL)
	}
	return rows, nil
}
