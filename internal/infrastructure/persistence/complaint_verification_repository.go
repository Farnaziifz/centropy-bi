package persistence

import (
	"context"

	"github.com/google/uuid"

	"centropy-affilate/ent"
	entverification "centropy-affilate/ent/complaintverification"
	"centropy-affilate/internal/domain/complaint"
	apperrors "centropy-affilate/pkg/errors"
)

type ComplaintVerificationRepository struct {
	client *ent.Client
}

func NewComplaintVerificationRepository(client *ent.Client) *ComplaintVerificationRepository {
	return &ComplaintVerificationRepository{client: client}
}

func (r *ComplaintVerificationRepository) Upsert(ctx context.Context, v *complaint.Verification) error {
	err := r.client.ComplaintVerification.Create().
		SetExternalUserID(v.ExternalUserID).
		SetComplaintAt(v.ComplaintAt).
		SetIsGenuine(v.IsGenuine).
		SetReasoning(v.Reasoning).
		SetVerifiedAt(v.VerifiedAt).
		OnConflictColumns(entverification.FieldExternalUserID).
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return apperrors.Wrap(apperrors.KindUnknown, "upsert complaint verification", err)
	}
	return nil
}

func (r *ComplaintVerificationRepository) FindByExternalUserID(ctx context.Context, externalUserID uuid.UUID) (*complaint.Verification, error) {
	row, err := r.client.ComplaintVerification.Query().
		Where(entverification.ExternalUserIDEQ(externalUserID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, apperrors.NotFound("verification not found")
		}
		return nil, apperrors.Wrap(apperrors.KindUnknown, "find complaint verification", err)
	}
	return toDomainVerification(row), nil
}

func (r *ComplaintVerificationRepository) List(ctx context.Context) ([]complaint.Verification, error) {
	rows, err := r.client.ComplaintVerification.Query().All(ctx)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "list complaint verifications", err)
	}
	out := make([]complaint.Verification, len(rows))
	for i, row := range rows {
		out[i] = *toDomainVerification(row)
	}
	return out, nil
}

func toDomainVerification(row *ent.ComplaintVerification) *complaint.Verification {
	return &complaint.Verification{
		ExternalUserID: row.ExternalUserID,
		ComplaintAt:    row.ComplaintAt,
		IsGenuine:      row.IsGenuine,
		Reasoning:      row.Reasoning,
		VerifiedAt:     row.VerifiedAt,
	}
}
