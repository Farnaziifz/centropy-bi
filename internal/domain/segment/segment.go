// Package segment defines the six customer segments from the "باشگاه
// مشتریان سنتروپی" roadmap (loyalty-club-roadmap.html) — newcomer, cold,
// hero, at-risk, churned, one-time — and the read model the admin API
// serves. The classification rule itself lives on the AlefGym side (see
// internal/infrastructure/alefgym/segment_repository.go) because, per the
// roadmap, every input it needs (Users, Orders, Courses.ExpiredAt) already
// exists there; this service adds no new fields to compute it.
package segment

import (
	"time"

	"github.com/google/uuid"
)

type Segment string

const (
	// Newcomer: registered <=14 days ago, zero completed orders yet.
	// Goal: activate the first purchase before the lead goes cold.
	Newcomer Segment = "NEWCOMER"
	// Cold: registered >14 days ago, still zero completed orders.
	// Goal: find out why they haven't bought before offering a discount.
	Cold Segment = "COLD"
	// Hero: at least 2 completed orders and a currently active course.
	// Goal: protect margin, turn them into a referrer.
	Hero Segment = "HERO"
	// AtRisk: a course expired within the last 30 days, not yet renewed.
	// Goal: highest-ROI segment — a known customer who just needs a nudge.
	AtRisk Segment = "AT_RISK"
	// Churned: a course expired more than 30 days ago, no completed order since.
	// Goal: the largest dormant-revenue opportunity — win-back campaign.
	Churned Segment = "CHURNED"
	// OneTime: exactly one completed order in the customer's entire history.
	// Goal: convert the first purchase into a second one.
	OneTime Segment = "ONE_TIME"
)

// All is the fixed display order used by the roadmap and, by extension,
// every summary/report in this service.
var All = []Segment{Newcomer, Cold, Hero, AtRisk, Churned, OneTime}

// Customer is the read-model row for a single member of a segment —
// intentionally thin (just enough for an admin list view); the full
// customer record is available via GET /admin/customers.
type Customer struct {
	ExternalUserID  uuid.UUID
	Phone           string
	FirstName       string
	LastName        string
	CompletedOrders int
}

// Count pairs a segment with how many customers currently fall into it.
type Count struct {
	Segment Segment
	Count   int
}

// Summary is the whole-book breakdown served by GET /admin/segments.
type Summary struct {
	Counts         []Count
	TotalCustomers int
}

// NonPurchaser is a customer who registered but has never completed a
// single order — the union of Newcomer and Cold, without splitting on the
// 14-day cutoff, for the plain "who signed up and never bought" report.
type NonPurchaser struct {
	ExternalUserID uuid.UUID
	Phone          string
	FirstName      string
	LastName       string
	RegisteredAt   time.Time
}

// MonthlySignups pairs a signup month ("2026-08") with how many of that
// month's registrants still have zero completed orders as of now.
type MonthlySignups struct {
	Month string
	Count int
}
