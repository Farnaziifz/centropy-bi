// Package renewal answers a specific, literal question: which customers
// received their last training/diet program more than N days ago and have
// bought nothing since. It's distinct from segment.Churned (which keys off
// Courses.ExpiredAt, the plan's paid-for end date, with a 30-day cutoff) —
// this keys off Courses.CreatedAt, the date the program itself was
// delivered, with a caller-chosen cutoff (default 50 days, per the request
// this was built for).
package renewal

import (
	"time"

	"github.com/google/uuid"
)

// OverdueCustomer is a customer whose most recent delivered program is
// older than the threshold, with zero completed orders since that program.
type OverdueCustomer struct {
	ExternalUserID       uuid.UUID
	Phone                string
	FirstName            string
	LastName             string
	LastProgramAt        time.Time
	DaysSinceLastProgram int
	CompletedOrdersTotal int
}
