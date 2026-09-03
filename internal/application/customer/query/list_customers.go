package query

import (
	"context"

	"centropy-affilate/internal/domain/customer"
)

type ListCustomersQuery struct{}

type ListCustomersHandler struct {
	repo customer.Repository
}

func NewListCustomersHandler(repo customer.Repository) *ListCustomersHandler {
	return &ListCustomersHandler{repo: repo}
}

func (h *ListCustomersHandler) Handle(ctx context.Context, _ ListCustomersQuery) ([]customer.Customer, error) {
	return h.repo.List(ctx)
}
