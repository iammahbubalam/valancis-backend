package sqlcrepo

import (
	"context"
	"encoding/json"
	"valancis-backend/db/sqlc"
	"valancis-backend/internal/domain"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type ContentRepository interface {
	GetContentByKey(ctx context.Context, key string) (*domain.ContentBlock, error)
	GetActiveContent(ctx context.Context, key string) (*domain.ContentBlock, error)
	UpsertContent(ctx context.Context, key string, content interface{}) (*domain.ContentBlock, error)
	UpdateSchedule(ctx context.Context, key string, isActive bool, startAt, endAt *time.Time) error
}

type contentRepository struct {
	q *sqlc.Queries
}

func NewContentRepository(db sqlc.DBTX) ContentRepository {
	return &contentRepository{
		q: sqlc.New(db),
	}
}

func (r *contentRepository) GetContentByKey(ctx context.Context, key string) (*domain.ContentBlock, error) {
	content, err := r.q.GetContentBlockByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	return mapToContentBlock(content), nil
}

// GetActiveContent returns content only if it's active and within schedule.
// L9: Uses DB-side filtering for performance with partial index.
func (r *contentRepository) GetActiveContent(ctx context.Context, key string) (*domain.ContentBlock, error) {
	content, err := r.q.GetActiveContentBlock(ctx, key)
	if err != nil {
		return nil, err
	}
	return mapToContentBlock(content), nil
}

func (r *contentRepository) UpsertContent(ctx context.Context, key string, data interface{}) (*domain.ContentBlock, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	content, err := r.q.UpsertContentBlock(ctx, sqlc.UpsertContentBlockParams{
		SectionKey: key,
		Content:    bytes,
	})
	if err != nil {
		return nil, err
	}
	return mapToContentBlock(content), nil
}

// UpdateSchedule updates the scheduling fields for a content block.
// L9: Separate method for schedule updates to avoid content overwrites.
func (r *contentRepository) UpdateSchedule(ctx context.Context, key string, isActive bool, startAt, endAt *time.Time) error {
	var pgStartAt, pgEndAt pgtype.Timestamp
	if startAt != nil {
		pgStartAt = pgtype.Timestamp{Time: *startAt, Valid: true}
	}
	if endAt != nil {
		pgEndAt = pgtype.Timestamp{Time: *endAt, Valid: true}
	}

	return r.q.UpdateContentBlockSchedule(ctx, sqlc.UpdateContentBlockScheduleParams{
		SectionKey: key,
		StartAt:    pgStartAt,
		EndAt:      pgEndAt,
		IsActive:   &isActive,
	})
}

// mapToContentBlock converts SQLC model to domain model.
// L9: Single mapping function for consistency.
func mapToContentBlock(c sqlc.ContentBlock) *domain.ContentBlock {
	block := &domain.ContentBlock{
		ID:         uuidToString(c.ID),
		SectionKey: c.SectionKey,
		Content:    domain.RawJSON(c.Content),
		UpdatedAt:  pgtimestamptzToTime(c.UpdatedAt),
	}

	// Map scheduling fields (handle nil pointers from DB)
	if c.IsActive != nil {
		block.IsActive = *c.IsActive
	} else {
		block.IsActive = true // Default to active if null
	}

	if c.StartAt.Valid {
		block.StartAt = &c.StartAt.Time
	}
	if c.EndAt.Valid {
		block.EndAt = &c.EndAt.Time
	}

	return block
}

func pgtimestamptzToTime(t pgtype.Timestamptz) time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Time{}
}
