package adminuser

import "context"

// Repository is the write/read port for AdminUser — small enough that,
// unlike Customer/Ticket in the reference architecture, it doesn't need a
// separate CQRS read side.
type Repository interface {
	Create(ctx context.Context, u *AdminUser) error
	FindByEmail(ctx context.Context, email string) (*AdminUser, error)
}
