package alefgym

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"centropy-affilate/internal/domain/renewal"
	apperrors "centropy-affilate/pkg/errors"
)

// RenewalRepository answers "who received their last program more than N
// days ago and never bought again" straight off COURSE.Courses and
// SALE.Orders — see the renewal package doc for how this differs from the
// segment package's Churned rule.
type RenewalRepository struct {
	db              *sql.DB
	excludedUserIDs []uuid.UUID
}

func NewRenewalRepository(db *sql.DB, excludedUserIDs []uuid.UUID) *RenewalRepository {
	return &RenewalRepository{db: db, excludedUserIDs: excludedUserIDs}
}

func (r *RenewalRepository) excludedAsText() []string {
	out := make([]string, len(r.excludedUserIDs))
	for i, id := range r.excludedUserIDs {
		out[i] = id.String()
	}
	return out
}

func (r *RenewalRepository) ListOverdueWithoutRepurchase(ctx context.Context, minDays int) ([]renewal.OverdueCustomer, error) {
	if minDays <= 0 {
		minDays = 50
	}

	query := `
		WITH last_course AS (
			SELECT "UserID", MAX("CreatedAt") AS last_program_at
			FROM "COURSE"."Courses"
			WHERE "IsDeleted" = false
			GROUP BY "UserID"
		)
		SELECT
			u."ID", u."PhoneNumber", u."FirstName", u."LastName", lc.last_program_at,
			(SELECT count(*) FROM "SALE"."Orders" o
			 WHERE o."UserID" = u."ID" AND o."Status" = 'COMPLETED' AND o."IsDeleted" = false) AS total_completed
		FROM "AUTHENTICATION"."Users" u
		JOIN last_course lc ON lc."UserID" = u."ID"
		WHERE u."IsDeleted" = false
		  AND u."ID"::text <> ALL($1::text[])
		  AND lc.last_program_at <= now() - ($2 * interval '1 day')
		  AND NOT EXISTS (
			SELECT 1 FROM "SALE"."Orders" o
			WHERE o."UserID" = u."ID" AND o."Status" = 'COMPLETED' AND o."IsDeleted" = false
			  AND o."CreatedAt" > lc.last_program_at
		  )
		ORDER BY lc.last_program_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, r.excludedAsText(), minDays)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "list overdue-without-repurchase customers", err)
	}
	defer rows.Close()

	now := time.Now()
	var out []renewal.OverdueCustomer
	for rows.Next() {
		var c renewal.OverdueCustomer
		var firstName, lastName sql.NullString
		if err := rows.Scan(&c.ExternalUserID, &c.Phone, &firstName, &lastName, &c.LastProgramAt, &c.CompletedOrdersTotal); err != nil {
			return nil, apperrors.Wrap(apperrors.KindUnknown, "scan overdue-without-repurchase customer", err)
		}
		c.FirstName = firstName.String
		c.LastName = lastName.String
		c.DaysSinceLastProgram = int(now.Sub(c.LastProgramAt).Hours() / 24)
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "iterate overdue-without-repurchase customers", err)
	}
	return out, nil
}
