package domain

import "context"

type SearchRepository interface {
	SearchProducts(ctx context.Context, query string, limit, offset int) ([]Product, int64, error)
}

type SearchUsecase interface {
	Search(ctx context.Context, query string, page, limit int) ([]Product, Pagination, error)
}
