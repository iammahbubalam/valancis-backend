package usecase

import (
	"context"
	"time"

	"valancis-backend/internal/domain"
)

type searchUsecase struct {
	searchRepo domain.SearchRepository
	timeout    time.Duration
}

func NewSearchUsecase(searchRepo domain.SearchRepository, timeout time.Duration) domain.SearchUsecase {
	return &searchUsecase{
		searchRepo: searchRepo,
		timeout:    timeout,
	}
}

func (u *searchUsecase) Search(ctx context.Context, query string, page, limit int) ([]domain.Product, domain.Pagination, error) {
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	products, total, err := u.searchRepo.SearchProducts(ctx, query, limit, offset)
	if err != nil {
		return nil, domain.Pagination{}, err
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	pagination := domain.Pagination{
		Page:       page,
		Limit:      limit,
		TotalItems: total,
		TotalPages: totalPages,
	}

	return products, pagination, nil
}
