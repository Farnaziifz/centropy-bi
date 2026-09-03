package query

import (
	"context"
	"fmt"
	"time"

	"centropy-affilate/internal/domain/segment"
	"centropy-affilate/internal/infrastructure/cache"
	apperrors "centropy-affilate/pkg/errors"
)

const listCacheTTL = 10 * time.Minute

type ListCustomersQuery struct {
	Segment segment.Segment
}

type ListCustomersHandler struct {
	repo  segment.Repository
	cache *cache.Cache
}

func NewListCustomersHandler(repo segment.Repository, c *cache.Cache) *ListCustomersHandler {
	return &ListCustomersHandler{repo: repo, cache: c}
}

func (h *ListCustomersHandler) Handle(ctx context.Context, q ListCustomersQuery) ([]segment.Customer, error) {
	if !isKnownSegment(q.Segment) {
		return nil, apperrors.InvalidInput("unknown segment")
	}

	cacheKey := fmt.Sprintf("segment:customers:%s", q.Segment)
	if h.cache != nil {
		if cached, err := cache.Get[[]segment.Customer](ctx, h.cache, cacheKey); err == nil {
			return cached, nil
		}
	}

	rows, err := h.repo.ListCustomers(ctx, q.Segment)
	if err != nil {
		return nil, err
	}

	if h.cache != nil {
		_ = cache.Set(ctx, h.cache, cacheKey, rows, listCacheTTL)
	}
	return rows, nil
}

func isKnownSegment(s segment.Segment) bool {
	for _, known := range segment.All {
		if known == s {
			return true
		}
	}
	return false
}
