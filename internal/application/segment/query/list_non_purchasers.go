package query

import (
	"context"

	"centropy-affilate/internal/domain/segment"
	"centropy-affilate/internal/infrastructure/cache"
)

const nonPurchasersCacheKey = "segment:non-purchasers"

type ListNonPurchasersQuery struct{}

type ListNonPurchasersHandler struct {
	repo  segment.Repository
	cache *cache.Cache
}

func NewListNonPurchasersHandler(repo segment.Repository, c *cache.Cache) *ListNonPurchasersHandler {
	return &ListNonPurchasersHandler{repo: repo, cache: c}
}

func (h *ListNonPurchasersHandler) Handle(ctx context.Context, _ ListNonPurchasersQuery) ([]segment.NonPurchaser, error) {
	if h.cache != nil {
		if cached, err := cache.Get[[]segment.NonPurchaser](ctx, h.cache, nonPurchasersCacheKey); err == nil {
			return cached, nil
		}
	}

	rows, err := h.repo.ListNonPurchasers(ctx)
	if err != nil {
		return nil, err
	}

	if h.cache != nil {
		_ = cache.Set(ctx, h.cache, nonPurchasersCacheKey, rows, listCacheTTL)
	}
	return rows, nil
}

// monthlyNonPurchasersCacheKey/TTL live alongside ListNonPurchasers rather
// than in their own file — same repository, same cache, same TTL as every
// other segment query (see summaryCacheTTL in get_summary.go).
const monthlyNonPurchasersCacheKey = "segment:non-purchasers:monthly"

type MonthlyNonPurchaserSignupsQuery struct{}

type MonthlyNonPurchaserSignupsHandler struct {
	repo  segment.Repository
	cache *cache.Cache
}

func NewMonthlyNonPurchaserSignupsHandler(repo segment.Repository, c *cache.Cache) *MonthlyNonPurchaserSignupsHandler {
	return &MonthlyNonPurchaserSignupsHandler{repo: repo, cache: c}
}

func (h *MonthlyNonPurchaserSignupsHandler) Handle(ctx context.Context, _ MonthlyNonPurchaserSignupsQuery) ([]segment.MonthlySignups, error) {
	if h.cache != nil {
		if cached, err := cache.Get[[]segment.MonthlySignups](ctx, h.cache, monthlyNonPurchasersCacheKey); err == nil {
			return cached, nil
		}
	}

	rows, err := h.repo.MonthlyNonPurchaserSignups(ctx)
	if err != nil {
		return nil, err
	}

	if h.cache != nil {
		_ = cache.Set(ctx, h.cache, monthlyNonPurchasersCacheKey, rows, listCacheTTL)
	}
	return rows, nil
}
