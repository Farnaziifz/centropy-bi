package alefgym

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	ptime "github.com/yaa110/go-persian-calendar"

	"centropy-affilate/internal/domain/segment"
	apperrors "centropy-affilate/pkg/errors"
)

// nonPurchaserFilter is the shared predicate both ListNonPurchasers and
// MonthlyNonPurchaserSignups build on: a non-deleted AlefGym user with zero
// completed orders to date — the union of Newcomer and Cold, without the
// 14-day split.
const nonPurchaserFilter = `
	u."IsDeleted" = false
	AND u."ID"::text <> ALL($1::text[])
	AND NOT EXISTS (
		SELECT 1 FROM "SALE"."Orders" o
		WHERE o."UserID" = u."ID" AND o."Status" = 'COMPLETED' AND o."IsDeleted" = false
	)
`

func (r *SegmentRepository) ListNonPurchasers(ctx context.Context) ([]segment.NonPurchaser, error) {
	query := `
		SELECT u."ID", u."PhoneNumber", u."FirstName", u."LastName", u."CreatedAt"
		FROM "AUTHENTICATION"."Users" u
		WHERE ` + nonPurchaserFilter + `
		ORDER BY u."CreatedAt" DESC
		LIMIT 5000
	`

	rows, err := r.db.QueryContext(ctx, query, r.excludedAsText())
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "list non-purchasers", err)
	}
	defer rows.Close()

	var out []segment.NonPurchaser
	for rows.Next() {
		var c segment.NonPurchaser
		var firstName, lastName sql.NullString
		if err := rows.Scan(&c.ExternalUserID, &c.Phone, &firstName, &lastName, &c.RegisteredAt); err != nil {
			return nil, apperrors.Wrap(apperrors.KindUnknown, "scan non-purchaser", err)
		}
		c.FirstName = firstName.String
		c.LastName = lastName.String
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "iterate non-purchasers", err)
	}
	return out, nil
}

// MonthlyNonPurchaserSignups groups non-purchasers by signup month — the
// data behind the "how many signups per month never bought" chart. Note
// this reflects *today's* purchase state: a signup counted in one month
// stays counted every month until they complete their first order, this
// isn't a historical snapshot of that month alone.
//
// Grouping happens in Go, not SQL: months are Jalali (Iran's calendar,
// what the admin panel displays), and a Gregorian calendar month doesn't
// map onto a single Jalali month, so date_trunc('month', ...) can't do
// this bucketing correctly. At AlefGym's current scale (~1k non-purchasers)
// pulling raw timestamps and bucketing them here is cheap.
func (r *SegmentRepository) MonthlyNonPurchaserSignups(ctx context.Context) ([]segment.MonthlySignups, error) {
	query := `
		SELECT u."CreatedAt"
		FROM "AUTHENTICATION"."Users" u
		WHERE ` + nonPurchaserFilter + `
	`

	rows, err := r.db.QueryContext(ctx, query, r.excludedAsText())
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "monthly non-purchaser signups", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	labels := make(map[string]string)
	for rows.Next() {
		var createdAt sql.NullTime
		if err := rows.Scan(&createdAt); err != nil {
			return nil, apperrors.Wrap(apperrors.KindUnknown, "scan non-purchaser signup date", err)
		}
		if !createdAt.Valid {
			continue
		}

		jt := ptime.New(createdAt.Time.In(ptime.Iran()))
		key := fmt.Sprintf("%04d-%02d", jt.Year(), int(jt.Month()))
		counts[key]++
		labels[key] = fmt.Sprintf("%s %d", jt.Month().String(), jt.Year())
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "iterate non-purchaser signup dates", err)
	}

	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]segment.MonthlySignups, len(keys))
	for i, k := range keys {
		out[i] = segment.MonthlySignups{Month: labels[k], Count: counts[k]}
	}
	return out, nil
}
