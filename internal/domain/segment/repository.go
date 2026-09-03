package segment

import "context"

// Repository is the read port onto the segment classification, implemented
// against the AlefGym production database (internal/infrastructure/alefgym).
type Repository interface {
	Summarize(ctx context.Context) (Summary, error)
	ListCustomers(ctx context.Context, seg Segment) ([]Customer, error)

	// ListNonPurchasers returns every registered customer with zero
	// completed orders to date (Newcomer + Cold combined).
	ListNonPurchasers(ctx context.Context) ([]NonPurchaser, error)
	// MonthlyNonPurchaserSignups returns, for every signup month, how many
	// of that month's registrants still have zero completed orders today.
	MonthlyNonPurchaserSignups(ctx context.Context) ([]MonthlySignups, error)
}
