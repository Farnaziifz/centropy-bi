package renewal

import "context"

type Repository interface {
	// ListOverdueWithoutRepurchase returns customers whose last delivered
	// program is more than minDays old and who have completed zero orders
	// since.
	ListOverdueWithoutRepurchase(ctx context.Context, minDays int) ([]OverdueCustomer, error)
}
