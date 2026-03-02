package sqlcrepo

import (
	"context"
	"fmt"
	"valancis-backend/internal/domain"
	"strconv"

	// Import the GENERATED SQLC package alias
	"valancis-backend/db/sqlc"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type couponRepository struct {
	q *sqlc.Queries
}

func NewCouponRepository(db sqlc.DBTX) domain.CouponRepository {
	return &couponRepository{
		q: sqlc.New(db),
	}
}

func (r *couponRepository) CreateCoupon(ctx context.Context, c *domain.Coupon) error {
	// Domain uses int, SQLC expects *int32
	usageLimit := int32(c.UsageLimit)
	// Domain uses bool, SQLC expects *bool
	isActive := c.IsActive

	params := sqlc.CreateCouponParams{
		Code:       c.Code,
		Type:       c.Type,
		UsageLimit: &usageLimit,
		IsActive:   &isActive,
	}

	// Convert Value - This is required, don't ignore errors
	val, err := Float64ToNumeric(c.Value)
	if err != nil {
		return fmt.Errorf("invalid coupon value: %w", err)
	}
	params.Value = val

	// Convert MinSpend (optional, default 0)
	minSpend, _ := Float64ToNumeric(c.MinSpend)
	params.MinSpend = minSpend

	if c.StartAt != nil {
		params.StartAt = pgtype.Timestamp{Time: *c.StartAt, Valid: true}
	}
	if c.ExpiresAt != nil {
		params.ExpiresAt = pgtype.Timestamp{Time: *c.ExpiresAt, Valid: true}
	}

	result, err := r.q.CreateCoupon(ctx, params)
	if err != nil {
		return err
	}

	// L9: Set the ID from the database response
	c.ID = uuid.UUID(result.ID.Bytes)
	c.CreatedAt = result.CreatedAt.Time
	return nil
}

func (r *couponRepository) GetCouponByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	c, err := r.q.GetCouponByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toDomainCoupon(c), nil
}

func (r *couponRepository) ValidateCoupon(ctx context.Context, code string, cartTotal float64) (*domain.CouponValidationResult, error) {
	total, _ := Float64ToNumeric(cartTotal)

	res, err := r.q.ValidateCoupon(ctx, sqlc.ValidateCouponParams{
		Code:      code,
		CartTotal: total,
	})
	if err != nil {
		return nil, err
	}

	return &domain.CouponValidationResult{
		ID:               uuid.UUID(res.ID.Bytes),
		Code:             res.Code,
		Type:             res.Type,
		Value:            NumericToFloat64(res.Value),
		MinSpend:         NumericToFloat64(res.MinSpend),
		ValidationStatus: res.ValidationStatus,
	}, nil
}

func (r *couponRepository) ListCoupons(ctx context.Context, limit, offset int) ([]domain.Coupon, error) {
	coupons, err := r.q.ListCoupons(ctx, sqlc.ListCouponsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	var result []domain.Coupon
	for _, c := range coupons {
		result = append(result, *toDomainCoupon(c))
	}
	return result, nil
}

func (r *couponRepository) IncrementCouponUsage(ctx context.Context, id uuid.UUID) error {
	// Convert uuid.UUID -> pgtype.UUID
	pgUUID := pgtype.UUID{Bytes: id, Valid: true}
	return r.q.IncrementCouponUsage(ctx, pgUUID)
}

func (r *couponRepository) DeleteCoupon(ctx context.Context, id uuid.UUID) error {
	pgUUID := pgtype.UUID{Bytes: id, Valid: true}
	return r.q.DeleteCoupon(ctx, pgUUID)
}

func (r *couponRepository) GetCouponByID(ctx context.Context, id uuid.UUID) (*domain.Coupon, error) {
	pgUUID := pgtype.UUID{Bytes: id, Valid: true}
	c, err := r.q.GetCouponByID(ctx, pgUUID)
	if err != nil {
		return nil, err
	}
	return toDomainCoupon(c), nil
}

func (r *couponRepository) UpdateCoupon(ctx context.Context, c *domain.Coupon) error {
	usageLimit := int32(c.UsageLimit)
	val, _ := Float64ToNumeric(c.Value)
	min, _ := Float64ToNumeric(c.MinSpend)

	var startAt, expiresAt pgtype.Timestamp
	if c.StartAt != nil {
		startAt = pgtype.Timestamp{Time: *c.StartAt, Valid: true}
	}
	if c.ExpiresAt != nil {
		expiresAt = pgtype.Timestamp{Time: *c.ExpiresAt, Valid: true}
	}

	return r.q.UpdateCoupon(ctx, sqlc.UpdateCouponParams{
		ID:         pgtype.UUID{Bytes: c.ID, Valid: true},
		Code:       c.Code,
		Type:       c.Type,
		Value:      val,
		MinSpend:   min,
		UsageLimit: &usageLimit,
		StartAt:    startAt,
		ExpiresAt:  expiresAt,
		IsActive:   &c.IsActive,
	})
}

func (r *couponRepository) CountCoupons(ctx context.Context) (int64, error) {
	return r.q.CountCoupons(ctx)
}

// Helpers
func toDomainCoupon(c sqlc.Coupon) *domain.Coupon {
	// SQLC model has *int32 for UsageLimit
	var limit, used int
	if c.UsageLimit != nil {
		limit = int(*c.UsageLimit)
	}
	if c.UsedCount != nil {
		used = int(*c.UsedCount)
	}
	var active bool
	if c.IsActive != nil {
		active = *c.IsActive
	}

	return &domain.Coupon{
		ID:         uuid.UUID(c.ID.Bytes),
		Code:       c.Code,
		Type:       c.Type,
		Value:      NumericToFloat64(c.Value),
		MinSpend:   NumericToFloat64(c.MinSpend),
		UsageLimit: limit,
		UsedCount:  used,
		StartAt:    toTimePtr(c.StartAt),
		ExpiresAt:  toTimePtr(c.ExpiresAt),
		IsActive:   active,
		CreatedAt:  c.CreatedAt.Time,
	}
}

func toTimePtr(t pgtype.Timestamp) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}

// NumericToFloat64 converts pgtype.Numeric to float64
func NumericToFloat64(n pgtype.Numeric) float64 {
	f, _ := n.Float64Value()
	return f.Float64
}

// Float64ToNumeric converts float64 to pgtype.Numeric
// L9: Use strconv.FormatFloat for zero-allocation conversion
func Float64ToNumeric(f float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	// Convert float64 to string with 2 decimal precision
	str := strconv.FormatFloat(f, 'f', 2, 64)
	err := n.Scan(str)
	if err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}
