package persistence

import (
	"context"

	"github.com/google/uuid"

	"centropy-affilate/ent"
	entanalysis "centropy-affilate/ent/customeranalysis"
	"centropy-affilate/internal/domain/analysis"
	apperrors "centropy-affilate/pkg/errors"
)

type AnalysisRepository struct {
	client *ent.Client
}

func NewAnalysisRepository(client *ent.Client) *AnalysisRepository {
	return &AnalysisRepository{client: client}
}

func (r *AnalysisRepository) Upsert(ctx context.Context, a *analysis.CustomerAnalysis) error {
	create := r.client.CustomerAnalysis.Create().
		SetExternalUserID(a.ExternalUserID).
		SetCategory(entanalysis.Category(a.Category)).
		SetSummary(a.Summary).
		SetConfidence(entanalysis.Confidence(a.Confidence)).
		SetAnalyzedAt(a.AnalyzedAt).
		OnConflictColumns(entanalysis.FieldExternalUserID).
		UpdateNewValues()

	if a.LastMessageAt != nil {
		create = create.SetLastMessageAt(*a.LastMessageAt)
	}

	if err := create.Exec(ctx); err != nil {
		return apperrors.Wrap(apperrors.KindUnknown, "upsert customer analysis", err)
	}
	return nil
}

func (r *AnalysisRepository) FindByExternalUserID(ctx context.Context, externalUserID uuid.UUID) (*analysis.CustomerAnalysis, error) {
	row, err := r.client.CustomerAnalysis.Query().
		Where(entanalysis.ExternalUserIDEQ(externalUserID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, apperrors.NotFound("analysis not found")
		}
		return nil, apperrors.Wrap(apperrors.KindUnknown, "find customer analysis", err)
	}
	return toDomainAnalysis(row), nil
}

func (r *AnalysisRepository) List(ctx context.Context) ([]analysis.CustomerAnalysis, error) {
	rows, err := r.client.CustomerAnalysis.Query().All(ctx)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnknown, "list customer analyses", err)
	}
	out := make([]analysis.CustomerAnalysis, len(rows))
	for i, row := range rows {
		out[i] = *toDomainAnalysis(row)
	}
	return out, nil
}

func toDomainAnalysis(row *ent.CustomerAnalysis) *analysis.CustomerAnalysis {
	a := &analysis.CustomerAnalysis{
		ExternalUserID: row.ExternalUserID,
		Category:       analysis.Category(row.Category),
		Summary:        row.Summary,
		Confidence:     string(row.Confidence),
		AnalyzedAt:     row.AnalyzedAt,
	}
	if row.LastMessageAt != nil {
		t := *row.LastMessageAt
		a.LastMessageAt = &t
	}
	return a
}
