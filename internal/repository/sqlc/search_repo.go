package sqlcrepo

import (
	"context"
	"encoding/json"

	"valancis-backend/db/sqlc"
	"valancis-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type searchRepository struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewSearchRepository(db *pgxpool.Pool) domain.SearchRepository {
	return &searchRepository{
		db: db,
		q:  sqlc.New(db),
	}
}

func (r *searchRepository) SearchProducts(ctx context.Context, query string, limit, offset int) ([]domain.Product, int64, error) {
	// Get total count
	count, err := r.q.CountSearchProducts(ctx, sqlc.CountSearchProductsParams{
		Query:    query,
		IsActive: boolPtr(true),
	})
	if err != nil {
		return nil, 0, err
	}

	if count == 0 {
		return []domain.Product{}, 0, nil
	}

	// Get products
	rows, err := r.q.SearchProducts(ctx, sqlc.SearchProductsParams{
		Query:    query,
		Limit:    int32(limit),
		Offset:   int32(offset),
		IsActive: boolPtr(true),
	})
	if err != nil {
		return nil, 0, err
	}

	products := make([]domain.Product, len(rows))
	for i, row := range rows {
		products[i] = mapSearchRowToProduct(row)
	}

	return products, count, nil
}

func mapSearchRowToProduct(row sqlc.SearchProductsRow) domain.Product {
	var images []string
	if len(row.Media) > 0 {
		_ = json.Unmarshal(row.Media, &images)
	}

	return domain.Product{
		ID:        uuidToString(row.ID),
		Name:      row.Name,
		Slug:      row.Slug,
		BasePrice: numericToFloat64(row.BasePrice),
		SalePrice: numericToFloat64Ptr(row.SalePrice),
		// Stock & SKU moved to variants
		IsFeatured: row.IsFeatured,
		IsActive:   row.IsActive,
		Images:     images,
		CreatedAt:  pgtimeToTime(row.CreatedAt),
		UpdatedAt:  pgtimeToTime(row.UpdatedAt),
	}
}

// Helpers
func boolPtr(b bool) *bool { return &b }
