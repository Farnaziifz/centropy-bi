package alefgym

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"centropy-affilate/internal/domain/analysis"
	apperrors "centropy-affilate/pkg/errors"
)

// MessageSource reads one customer's own chat/ticket messages (both
// tables, "STUDENT"-authored only — never a coach/support reply) for the
// AI analysis job. Capped at 300 messages, oldest kept, which comfortably
// covers every real customer seen so far (the busiest is nowhere close)
// while bounding worst-case LLM input size.
type MessageSource struct {
	db *sql.DB
}

func NewMessageSource(db *sql.DB) *MessageSource {
	return &MessageSource{db: db}
}

func (s *MessageSource) FetchCustomerMessages(ctx context.Context, externalUserID uuid.UUID, since *time.Time) ([]analysis.CustomerMessage, error) {
	var sinceVal any
	if since != nil {
		sinceVal = *since
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT content, created_at FROM (
			SELECT "Content" AS content, "CreatedAt" AS created_at
			FROM "COMMUNICATION"."Messages"
			WHERE "UserID" = $1 AND "UserType" = 'STUDENT' AND "IsDeleted" = false
			UNION ALL
			SELECT "Content", "CreatedAt"
			FROM "COMMUNICATION"."TicketMessages"
			WHERE "UserID" = $1 AND "UserType" = 'STUDENT' AND "IsDeleted" = false
		) t
		WHERE $2::timestamp IS NULL OR created_at > $2::timestamp
		ORDER BY created_at ASC
		LIMIT 300
	`, externalUserID, sinceVal)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "fetch customer messages", err)
	}
	defer rows.Close()

	var out []analysis.CustomerMessage
	for rows.Next() {
		var m analysis.CustomerMessage
		if err := rows.Scan(&m.Content, &m.CreatedAt); err != nil {
			return nil, apperrors.Wrap(apperrors.KindUnknown, "scan customer message", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "iterate customer messages", err)
	}
	return out, nil
}

// LatestMessageAt answers "who in this batch has a message newer than X"
// in one round trip instead of one per candidate — see the interface doc
// on why that matters for a 400+-candidate batch run.
func (s *MessageSource) LatestMessageAt(ctx context.Context, externalUserIDs []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	if len(externalUserIDs) == 0 {
		return map[uuid.UUID]time.Time{}, nil
	}

	ids := make([]string, len(externalUserIDs))
	for i, id := range externalUserIDs {
		ids[i] = id.String()
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, MAX(created_at) FROM (
			SELECT "UserID" AS user_id, "CreatedAt" AS created_at
			FROM "COMMUNICATION"."Messages"
			WHERE "UserType" = 'STUDENT' AND "IsDeleted" = false AND "UserID"::text = ANY($1::text[])
			UNION ALL
			SELECT "UserID", "CreatedAt"
			FROM "COMMUNICATION"."TicketMessages"
			WHERE "UserType" = 'STUDENT' AND "IsDeleted" = false AND "UserID"::text = ANY($1::text[])
		) t
		GROUP BY user_id
	`, ids)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "fetch latest message timestamps", err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID]time.Time, len(externalUserIDs))
	for rows.Next() {
		var id uuid.UUID
		var latest time.Time
		if err := rows.Scan(&id, &latest); err != nil {
			return nil, apperrors.Wrap(apperrors.KindUnknown, "scan latest message timestamp", err)
		}
		out[id] = latest
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "iterate latest message timestamps", err)
	}
	return out, nil
}
