package alefgym

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"centropy-affilate/internal/domain/complaint"
	apperrors "centropy-affilate/pkg/errors"
)

// ComplaintRepository mines COMMUNICATION.Messages and TicketMessages for
// customer-authored complaints about late program delivery, then checks
// purchase history around each complaint. It's a separate repository from
// SegmentRepository — different source tables (chat/tickets vs
// orders/courses) and a fundamentally different kind of signal (a
// text-matched opinion, not a computed fact) — even though both read the
// same AlefGym database.
type ComplaintRepository struct {
	db              *sql.DB
	excludedUserIDs []uuid.UUID
}

func NewComplaintRepository(db *sql.DB, excludedUserIDs []uuid.UUID) *ComplaintRepository {
	return &ComplaintRepository{db: db, excludedUserIDs: excludedUserIDs}
}

func (r *ComplaintRepository) excludedAsText() []string {
	out := make([]string, len(r.excludedUserIDs))
	for i, id := range r.excludedUserIDs {
		out[i] = id.String()
	}
	return out
}

// delayKeywords matches the same "تأخیر شدید در تحویل برنامه" theme found
// by manually reading tickets/chat during the churn analysis: complaints
// that the training/diet program itself was late, not merely a support
// request. This is a keyword net, not proof — see package doc. col must be
// a fully-qualified column reference (e.g. `m."Content"`); it can't be a
// SELECT-list alias since it's used in that same SELECT's WHERE clause.
func delayKeywords(col string) string {
	return `(
		` + col + ` ILIKE '%دیر%برنامه%' OR ` + col + ` ILIKE '%برنامه%دیر%' OR
		` + col + ` ILIKE '%هنوز برنامه%' OR ` + col + ` ILIKE '%منتظر برنامه%' OR
		` + col + ` ILIKE '%تاخیر%برنامه%' OR ` + col + ` ILIKE '%برنامه%تاخیر%' OR
		` + col + ` ILIKE '%دیرکرد%'
	)`
}

func (r *ComplaintRepository) ListDelayedProgramComplainers(ctx context.Context) ([]complaint.DelayedProgramComplainer, error) {
	query := `
		WITH complaints AS (
			SELECT m."UserID" AS user_id, m."CreatedAt" AS complaint_at, m."Content" AS content
			FROM "COMMUNICATION"."Messages" m
			WHERE m."UserType" = 'STUDENT' AND m."IsDeleted" = false AND ` + delayKeywords(`m."Content"`) + `
			UNION ALL
			SELECT tm."UserID", tm."CreatedAt", tm."Content"
			FROM "COMMUNICATION"."TicketMessages" tm
			WHERE tm."UserType" = 'STUDENT' AND tm."IsDeleted" = false AND ` + delayKeywords(`tm."Content"`) + `
		),
		-- one row per user: their most recent delay complaint, since that's
		-- the one whose "did they ever buy again after this" question matters.
		latest AS (
			SELECT DISTINCT ON (user_id) user_id, complaint_at, content
			FROM complaints
			ORDER BY user_id, complaint_at DESC
		),
		classified AS (
			SELECT
				u."ID" AS user_id,
				u."PhoneNumber" AS phone,
				u."FirstName" AS first_name,
				u."LastName" AS last_name,
				l.complaint_at,
				left(l.content, 300) AS excerpt,
				(SELECT count(*) FROM "SALE"."Orders" o
				 WHERE o."UserID" = u."ID" AND o."Status" = 'COMPLETED' AND o."IsDeleted" = false
				   AND o."CreatedAt" <= l.complaint_at) AS orders_before,
				(SELECT count(*) FROM "SALE"."Orders" o
				 WHERE o."UserID" = u."ID" AND o."Status" = 'COMPLETED' AND o."IsDeleted" = false
				   AND o."CreatedAt" > l.complaint_at) AS orders_after
			FROM latest l
			JOIN "AUTHENTICATION"."Users" u ON u."ID" = l.user_id
			WHERE u."IsDeleted" = false AND u."ID"::text <> ALL($1::text[])
		)
		SELECT user_id, phone, first_name, last_name, complaint_at, excerpt, orders_before
		FROM classified
		WHERE orders_after = 0
		ORDER BY complaint_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, r.excludedAsText())
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "list delayed-program complainers", err)
	}
	defer rows.Close()

	var out []complaint.DelayedProgramComplainer
	for rows.Next() {
		var c complaint.DelayedProgramComplainer
		var firstName, lastName sql.NullString
		if err := rows.Scan(
			&c.ExternalUserID, &c.Phone, &firstName, &lastName,
			&c.ComplaintAt, &c.ComplaintExcerpt, &c.CompletedOrdersBefore,
		); err != nil {
			return nil, apperrors.Wrap(apperrors.KindUnknown, "scan delayed-program complainer", err)
		}
		c.FirstName = firstName.String
		c.LastName = lastName.String
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "iterate delayed-program complainers", err)
	}
	return out, nil
}
