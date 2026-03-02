package usecase

import (
	"context"
	"fmt"
	"valancis-backend/internal/domain"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CouponUsecase handles admin coupon management operations.
// L9: Follows single responsibility principle - focuses only on coupon CRUD.
type CouponUsecase struct {
	couponRepo domain.CouponRepository
}

// NewCouponUsecase creates a new CouponUsecase instance.
func NewCouponUsecase(couponRepo domain.CouponRepository) *CouponUsecase {
	return &CouponUsecase{
		couponRepo: couponRepo,
	}
}

// CreateCouponRequest represents the input for creating a coupon.
type CreateCouponRequest struct {
	Code       string  `json:"code"`
	Type       string  `json:"type"` // "percentage" or "fixed"
	Value      float64 `json:"value"`
	MinSpend   float64 `json:"minSpend"`
	UsageLimit int     `json:"usageLimit"`
	StartAt    string  `json:"startAt"`   // ISO8601 format
	ExpiresAt  string  `json:"expiresAt"` // ISO8601 format
	IsActive   bool    `json:"isActive"`
}

// CreateCoupon creates a new coupon with validation.
// L9: Input validation at usecase layer, not handler.
func (uc *CouponUsecase) CreateCoupon(ctx context.Context, req CreateCouponRequest) (*domain.Coupon, error) {
	// Validation: Code is required and must be uppercase
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		return nil, fmt.Errorf("coupon code is required")
	}

	// Validation: Type must be valid
	if req.Type != "percentage" && req.Type != "fixed" {
		return nil, fmt.Errorf("coupon type must be 'percentage' or 'fixed'")
	}

	// Validation: Value must be positive
	if req.Value <= 0 {
		return nil, fmt.Errorf("coupon value must be greater than 0")
	}

	// Validation: Percentage cannot exceed 100
	if req.Type == "percentage" && req.Value > 100 {
		return nil, fmt.Errorf("percentage discount cannot exceed 100%%")
	}

	// Check for duplicate code
	existing, _ := uc.couponRepo.GetCouponByCode(ctx, code)
	if existing != nil {
		return nil, fmt.Errorf("coupon code '%s' already exists", code)
	}

	coupon := &domain.Coupon{
		Code:       code,
		Type:       req.Type,
		Value:      req.Value,
		MinSpend:   req.MinSpend,
		UsageLimit: req.UsageLimit,
		IsActive:   req.IsActive,
	}

	// Parse dates if provided
	if req.StartAt != "" {
		if t, err := parseISO8601(req.StartAt); err == nil {
			coupon.StartAt = &t
		}
	}
	if req.ExpiresAt != "" {
		if t, err := parseISO8601(req.ExpiresAt); err == nil {
			coupon.ExpiresAt = &t
		}
	}

	if err := uc.couponRepo.CreateCoupon(ctx, coupon); err != nil {
		return nil, fmt.Errorf("failed to create coupon: %w", err)
	}

	return coupon, nil
}

// ListCoupons returns paginated list of coupons.
func (uc *CouponUsecase) ListCoupons(ctx context.Context, limit, offset int) ([]domain.Coupon, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // L9: Cap for safety
	}

	coupons, err := uc.couponRepo.ListCoupons(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list coupons: %w", err)
	}

	total, err := uc.couponRepo.CountCoupons(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count coupons: %w", err)
	}

	return coupons, total, nil
}

// GetCoupon returns a single coupon by ID.
func (uc *CouponUsecase) GetCoupon(ctx context.Context, id string) (*domain.Coupon, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid coupon ID")
	}

	coupon, err := uc.couponRepo.GetCouponByID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("coupon not found")
	}

	return coupon, nil
}

// UpdateCouponRequest represents the input for updating a coupon.
type UpdateCouponRequest struct {
	Code       string  `json:"code"`
	Type       string  `json:"type"`
	Value      float64 `json:"value"`
	MinSpend   float64 `json:"minSpend"`
	UsageLimit int     `json:"usageLimit"`
	StartAt    string  `json:"startAt"`
	ExpiresAt  string  `json:"expiresAt"`
	IsActive   bool    `json:"isActive"`
}

// UpdateCoupon updates an existing coupon.
func (uc *CouponUsecase) UpdateCoupon(ctx context.Context, id string, req UpdateCouponRequest) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid coupon ID")
	}

	// Verify coupon exists
	existing, err := uc.couponRepo.GetCouponByID(ctx, uid)
	if err != nil {
		return fmt.Errorf("coupon not found")
	}

	// Validation
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		return fmt.Errorf("coupon code is required")
	}

	if req.Type != "percentage" && req.Type != "fixed" {
		return fmt.Errorf("coupon type must be 'percentage' or 'fixed'")
	}

	if req.Value <= 0 {
		return fmt.Errorf("coupon value must be greater than 0")
	}

	if req.Type == "percentage" && req.Value > 100 {
		return fmt.Errorf("percentage discount cannot exceed 100%%")
	}

	// Check for duplicate code (if changed)
	if code != existing.Code {
		dup, _ := uc.couponRepo.GetCouponByCode(ctx, code)
		if dup != nil {
			return fmt.Errorf("coupon code '%s' already exists", code)
		}
	}

	coupon := &domain.Coupon{
		ID:         uid,
		Code:       code,
		Type:       req.Type,
		Value:      req.Value,
		MinSpend:   req.MinSpend,
		UsageLimit: req.UsageLimit,
		IsActive:   req.IsActive,
	}

	if req.StartAt != "" {
		if t, err := parseISO8601(req.StartAt); err == nil {
			coupon.StartAt = &t
		}
	}
	if req.ExpiresAt != "" {
		if t, err := parseISO8601(req.ExpiresAt); err == nil {
			coupon.ExpiresAt = &t
		}
	}

	return uc.couponRepo.UpdateCoupon(ctx, coupon)
}

// DeleteCoupon deletes a coupon by ID.
func (uc *CouponUsecase) DeleteCoupon(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid coupon ID")
	}

	// Verify exists
	if _, err := uc.couponRepo.GetCouponByID(ctx, uid); err != nil {
		return fmt.Errorf("coupon not found")
	}

	return uc.couponRepo.DeleteCoupon(ctx, uid)
}

// parseISO8601 parses an ISO8601 date string.
func parseISO8601(s string) (time.Time, error) {
	// Try multiple formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date format")
}
