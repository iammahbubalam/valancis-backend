package usecase

import (
	"context"
	"valancis-backend/internal/domain"
	repo "valancis-backend/internal/repository/sqlc"
	"time"
)

type ContentUsecase interface {
	GetContent(ctx context.Context, key string) (*domain.ContentBlock, error)
	GetActiveContent(ctx context.Context, key string) (*domain.ContentBlock, error)
	UpsertContent(ctx context.Context, key string, content interface{}) (*domain.ContentBlock, error)
	UpdateSchedule(ctx context.Context, key string, isActive bool, startAt, endAt *time.Time) error
}

type contentUsecase struct {
	repo repo.ContentRepository
}

func NewContentUsecase(r repo.ContentRepository) ContentUsecase {
	return &contentUsecase{repo: r}
}

func (u *contentUsecase) GetContent(ctx context.Context, key string) (*domain.ContentBlock, error) {
	return u.repo.GetContentByKey(ctx, key)
}

// GetActiveContent returns content only if it's currently active and scheduled.
// L9: For public endpoints - respects scheduling.
func (u *contentUsecase) GetActiveContent(ctx context.Context, key string) (*domain.ContentBlock, error) {
	return u.repo.GetActiveContent(ctx, key)
}

func (u *contentUsecase) UpsertContent(ctx context.Context, key string, content interface{}) (*domain.ContentBlock, error) {
	return u.repo.UpsertContent(ctx, key, content)
}

// UpdateSchedule updates the scheduling fields for a content block.
// L9: Admin-only method for time-based content control.
func (u *contentUsecase) UpdateSchedule(ctx context.Context, key string, isActive bool, startAt, endAt *time.Time) error {
	return u.repo.UpdateSchedule(ctx, key, isActive, startAt, endAt)
}
