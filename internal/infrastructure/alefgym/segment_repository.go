package alefgym

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"centropy-affilate/internal/domain/segment"
	apperrors "centropy-affilate/pkg/errors"
)

// SegmentRepository classifies every AlefGym customer into one of the six
// loyalty-club segments, straight from loyalty-club-roadmap.html's rules.
// It deliberately stores nothing: every input (signup date, completed
// order count, latest course expiry) already lives in AlefGym, so there is
// no new data model to keep in sync — only a query to run on demand.
//
// Rule precedence (top-to-bottom, first match wins) resolves the cases the
// roadmap's prose leaves ambiguous — e.g. a customer with exactly one
// completed order whose course also expired 40 days ago matches both
// "one-time" and "churned" by the roadmap's plain-English rules. This
// repository always prefers the course-lifecycle segments (hero / at-risk
// / churned) over "one-time", since "what's happening with their current
// course" is the more actionable signal for a campaign than "how many
// orders they've ever placed".
type SegmentRepository struct {
	db              *sql.DB
	excludedUserIDs []uuid.UUID
	log             *slog.Logger
}

func NewSegmentRepository(db *sql.DB, excludedUserIDs []uuid.UUID, log *slog.Logger) *SegmentRepository {
	return &SegmentRepository{db: db, excludedUserIDs: excludedUserIDs, log: log}
}

// classifiedRows is the shared CTE both Summarize and ListCustomers build
// on: one row per non-deleted AlefGym user, with their completed-order
// count, latest course expiry, and the segment that falls out of it.
const classifiedRows = `
WITH agg AS (
	SELECT
		u."ID"          AS user_id,
		u."PhoneNumber" AS phone,
		u."FirstName"   AS first_name,
		u."LastName"    AS last_name,
		u."CreatedAt"   AS registered_at,
		COALESCE(o.completed_orders, 0) AS completed_orders,
		c.max_expired_at
	FROM "AUTHENTICATION"."Users" u
	LEFT JOIN (
		SELECT "UserID", COUNT(*) AS completed_orders
		FROM "SALE"."Orders"
		WHERE "Status" = 'COMPLETED' AND "IsDeleted" = false
		GROUP BY "UserID"
	) o ON o."UserID" = u."ID"
	LEFT JOIN (
		SELECT "UserID", MAX("ExpiredAt") AS max_expired_at
		FROM "COURSE"."Courses"
		WHERE "IsDeleted" = false AND "ExpiredAt" IS NOT NULL
		GROUP BY "UserID"
	) c ON c."UserID" = u."ID"
	WHERE u."IsDeleted" = false
	  AND u."ID"::text <> ALL($1::text[])
)
SELECT
	user_id, phone, first_name, last_name, completed_orders,
	CASE
		WHEN completed_orders = 0 AND registered_at >= now() - interval '14 days' THEN 'NEWCOMER'
		WHEN completed_orders = 0 THEN 'COLD'
		WHEN completed_orders >= 2 AND max_expired_at IS NOT NULL AND max_expired_at > now() THEN 'HERO'
		WHEN max_expired_at IS NOT NULL AND max_expired_at <= now() AND max_expired_at > now() - interval '30 days' THEN 'AT_RISK'
		WHEN max_expired_at IS NOT NULL AND max_expired_at <= now() - interval '30 days' THEN 'CHURNED'
		WHEN completed_orders = 1 THEN 'ONE_TIME'
		ELSE 'UNCLASSIFIED'
	END AS segment
FROM agg
`

func (r *SegmentRepository) excludedAsText() []string {
	out := make([]string, len(r.excludedUserIDs))
	for i, id := range r.excludedUserIDs {
		out[i] = id.String()
	}
	return out
}

func (r *SegmentRepository) Summarize(ctx context.Context) (segment.Summary, error) {
	query := fmt.Sprintf(`SELECT segment, count(*) FROM (%s) t GROUP BY segment`, classifiedRows)

	rows, err := r.db.QueryContext(ctx, query, r.excludedAsText())
	if err != nil {
		return segment.Summary{}, apperrors.Wrap(apperrors.KindUnknown, "summarize segments", err)
	}
	defer rows.Close()

	counted := make(map[segment.Segment]int, len(segment.All))
	total := 0
	for rows.Next() {
		var seg string
		var count int
		if err := rows.Scan(&seg, &count); err != nil {
			return segment.Summary{}, apperrors.Wrap(apperrors.KindUnknown, "scan segment count", err)
		}
		total += count
		if seg == "UNCLASSIFIED" {
			r.log.Warn("alefgym: customers fell through every segment rule", "count", count)
			continue
		}
		counted[segment.Segment(seg)] = count
	}
	if err := rows.Err(); err != nil {
		return segment.Summary{}, apperrors.Wrap(apperrors.KindUnknown, "iterate segment counts", err)
	}

	summary := segment.Summary{TotalCustomers: total}
	for _, seg := range segment.All {
		summary.Counts = append(summary.Counts, segment.Count{Segment: seg, Count: counted[seg]})
	}
	return summary, nil
}

func (r *SegmentRepository) ListCustomers(ctx context.Context, seg segment.Segment) ([]segment.Customer, error) {
	query := fmt.Sprintf(
		`SELECT user_id, phone, first_name, last_name, completed_orders
		 FROM (%s) t
		 WHERE segment = $2
		 ORDER BY completed_orders DESC
		 LIMIT 500`, classifiedRows)

	rows, err := r.db.QueryContext(ctx, query, r.excludedAsText(), string(seg))
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "list segment customers", err)
	}
	defer rows.Close()

	var out []segment.Customer
	for rows.Next() {
		var c segment.Customer
		var firstName, lastName sql.NullString
		if err := rows.Scan(&c.ExternalUserID, &c.Phone, &firstName, &lastName, &c.CompletedOrders); err != nil {
			return nil, apperrors.Wrap(apperrors.KindUnknown, "scan segment customer", err)
		}
		c.FirstName = firstName.String
		c.LastName = lastName.String
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "iterate segment customers", err)
	}
	return out, nil
}
