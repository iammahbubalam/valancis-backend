package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"valancis-backend/db/sqlc"
	"valancis-backend/pkg/cache"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// L9 Stats Usecase: Validation only, no business logic hardcoding
type StatsUsecase struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
	cache   cache.CacheService
}

func NewStatsUsecase(db *pgxpool.Pool, cache cache.CacheService) *StatsUsecase {
	return &StatsUsecase{
		db:      db,
		queries: sqlc.New(db),
		cache:   cache,
	}
}

// Helper to convert time.Time to pgtype.Timestamp
func timeToPgTimestamp(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: t, Valid: true}
}

// GetDailySales - L9: Validate inputs, frontend controls date range & pagination
func (uc *StatsUsecase) GetDailySales(ctx context.Context, start, end time.Time, limit, offset int32) ([]sqlc.GetDailySalesRow, error) {
	if end.Before(start) {
		return nil, errors.New("end date must be after start date")
	}
	if end.Sub(start) > 365*24*time.Hour {
		return nil, errors.New("date range cannot exceed 1 year (performance limit)")
	}
	if limit < 1 {
		limit = 30
	}
	if offset < 0 {
		offset = 0
	}

	cacheKey := fmt.Sprintf("stats:daily_sales:%s:%s:%d:%d", start.Format("2006-01-02"), end.Format("2006-01-02"), limit, offset)

	if val, found := uc.cache.Get(cacheKey); found {
		return val.([]sqlc.GetDailySalesRow), nil
	}

	rows, err := uc.queries.GetDailySales(ctx, sqlc.GetDailySalesParams{
		StartDate:   timeToPgTimestamp(start),
		EndDate:     timeToPgTimestamp(end),
		LimitCount:  limit,
		OffsetCount: offset,
	})
	if err != nil {
		return nil, err
	}

	uc.cache.Set(cacheKey, rows, 30*time.Minute)
	return rows, nil
}

// GetRevenueKPIs - L9: Parameterized KPIs
func (uc *StatsUsecase) GetRevenueKPIs(ctx context.Context, start, end time.Time) (*sqlc.GetRevenueKPIsRow, error) {
	if end.Before(start) {
		return nil, errors.New("end date must be after start date")
	}
	if end.Sub(start) > 365*24*time.Hour {
		return nil, errors.New("date range cannot exceed 1 year")
	}

	cacheKey := fmt.Sprintf("stats:kpis:%s:%s", start.Format("2006-01-02"), end.Format("2006-01-02"))

	if val, found := uc.cache.Get(cacheKey); found {
		kpis := val.(sqlc.GetRevenueKPIsRow)
		return &kpis, nil
	}

	kpis, err := uc.queries.GetRevenueKPIs(ctx, sqlc.GetRevenueKPIsParams{
		StartDate: timeToPgTimestamp(start),
		EndDate:   timeToPgTimestamp(end),
	})
	if err != nil {
		return nil, err
	}

	uc.cache.Set(cacheKey, kpis, 30*time.Minute)
	return &kpis, nil
}

// GetLowStockProducts - L9: Frontend controls threshold and limit
func (uc *StatsUsecase) GetLowStockProducts(ctx context.Context, threshold, limit int32) ([]sqlc.GetLowStockProductsRow, error) {
	if threshold < 0 {
		return nil, errors.New("threshold must be non-negative")
	}
	if limit < 1 || limit > 500 {
		return nil, errors.New("limit must be 1-500")
	}

	cacheKey := fmt.Sprintf("stats:low_stock:%d:%d", threshold, limit)

	if val, found := uc.cache.Get(cacheKey); found {
		return val.([]sqlc.GetLowStockProductsRow), nil
	}

	products, err := uc.queries.GetLowStockProducts(ctx, sqlc.GetLowStockProductsParams{
		Threshold:  threshold,
		LimitCount: limit,
	})
	if err != nil {
		return nil, err
	}

	uc.cache.Set(cacheKey, products, 5*time.Minute)
	return products, nil
}

// GetDeadStockProducts - L9: Frontend controls days and limit
func (uc *StatsUsecase) GetDeadStockProducts(ctx context.Context, days, limit int32) ([]sqlc.GetDeadStockProductsRow, error) {
	if days < 1 || days > 365 {
		return nil, errors.New("days must be 1-365")
	}
	if limit < 1 || limit > 500 {
		return nil, errors.New("limit must be 1-500")
	}

	cacheKey := fmt.Sprintf("stats:dead_stock:%d:%d", days, limit)

	if val, found := uc.cache.Get(cacheKey); found {
		return val.([]sqlc.GetDeadStockProductsRow), nil
	}

	products, err := uc.queries.GetDeadStockProducts(ctx, sqlc.GetDeadStockProductsParams{
		Days:       days,
		LimitCount: limit,
	})
	if err != nil {
		return nil, err
	}

	uc.cache.Set(cacheKey, products, 15*time.Minute)
	return products, nil
}

// GetTopSellingProducts - L9: Frontend controls date range and limit
func (uc *StatsUsecase) GetTopSellingProducts(ctx context.Context, start, end time.Time, limit int32) ([]sqlc.GetTopSellingProductsRow, error) {
	if end.Before(start) {
		return nil, errors.New("end date must be after start date")
	}
	if limit < 1 || limit > 500 {
		return nil, errors.New("limit must be 1-500")
	}

	// Log start of usecase
	fmt.Printf("GetTopSellingProductsUsecase: Start - start=%v, end=%v, limit=%d\n", start, end, limit)

	cacheKey := fmt.Sprintf("stats:top_products:%s:%s:%d", start.Format("2006-01-02"), end.Format("2006-01-02"), limit)

	if val, found := uc.cache.Get(cacheKey); found {
		fmt.Println("GetTopSellingProductsUsecase: Cache Hit")
		return val.([]sqlc.GetTopSellingProductsRow), nil
	}
	fmt.Println("GetTopSellingProductsUsecase: Cache Miss - Querying DB")

	products, err := uc.queries.GetTopSellingProducts(ctx, sqlc.GetTopSellingProductsParams{
		StartDate:  timeToPgTimestamp(start),
		EndDate:    timeToPgTimestamp(end),
		LimitCount: limit,
	})
	if err != nil {
		fmt.Printf("GetTopSellingProductsUsecase: DB Error - %v\n", err)
		return nil, err
	}

	fmt.Printf("GetTopSellingProductsUsecase: DB Success - Found %d items\n", len(products))

	uc.cache.Set(cacheKey, products, 30*time.Minute)
	return products, nil
}

// GetCustomerLTV - L9: Frontend controls date range and limit
func (uc *StatsUsecase) GetCustomerLTV(ctx context.Context, start, end time.Time, limit int32) ([]sqlc.GetCustomerLTVRow, error) {
	if end.Before(start) {
		return nil, errors.New("end date must be after start date")
	}
	if limit < 1 || limit > 500 {
		return nil, errors.New("limit must be 1-500")
	}

	cacheKey := fmt.Sprintf("stats:customer_ltv:%s:%s:%d", start.Format("2006-01-02"), end.Format("2006-01-02"), limit)

	if val, found := uc.cache.Get(cacheKey); found {
		return val.([]sqlc.GetCustomerLTVRow), nil
	}

	customers, err := uc.queries.GetCustomerLTV(ctx, sqlc.GetCustomerLTVParams{
		StartDate:  timeToPgTimestamp(start),
		EndDate:    timeToPgTimestamp(end),
		LimitCount: limit,
	})
	if err != nil {
		return nil, err
	}

	uc.cache.Set(cacheKey, customers, 30*time.Minute)
	return customers, nil
}

// GetCustomerRetention - L9: Frontend controls date range
func (uc *StatsUsecase) GetCustomerRetention(ctx context.Context, start, end time.Time) (*sqlc.GetCustomerRetentionRow, error) {
	if end.Before(start) {
		return nil, errors.New("end date must be after start date")
	}

	cacheKey := fmt.Sprintf("stats:retention:%s:%s", start.Format("2006-01-02"), end.Format("2006-01-02"))

	if val, found := uc.cache.Get(cacheKey); found {
		ret := val.(sqlc.GetCustomerRetentionRow)
		return &ret, nil
	}

	retention, err := uc.queries.GetCustomerRetention(ctx, sqlc.GetCustomerRetentionParams{
		StartDate: timeToPgTimestamp(start),
		EndDate:   timeToPgTimestamp(end),
	})
	if err != nil {
		return nil, err
	}

	uc.cache.Set(cacheKey, retention, 30*time.Minute)
	return &retention, nil
}
