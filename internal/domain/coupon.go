package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Coupon struct {
	ID         uuid.UUID  `json:"id"`
	Code       string     `json:"code"`
	Type       string     `json:"type"` // percentage, fixed
	Value      float64    `json:"value"`
	MinSpend   float64    `json:"minSpend"`
	UsageLimit int        `json:"usageLimit"`
	UsedCount  int        `json:"usedCount"`
	StartAt    *time.Time `json:"startAt"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	IsActive   bool       `json:"isActive"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type CouponValidationResult struct {
	ID               uuid.UUID `json:"id"`
	Code             string    `json:"code"`
	Type             string    `json:"type"`
	Value            float64   `json:"value"`
	MinSpend         float64   `json:"minSpend"`
	ValidationStatus string    `json:"validationStatus"` // valid, inactive, expired, etc.
}

type CouponRepository interface {
	CreateCoupon(ctx context.Context, coupon *Coupon) error
	GetCouponByCode(ctx context.Context, code string) (*Coupon, error)
	GetCouponByID(ctx context.Context, id uuid.UUID) (*Coupon, error)
	ValidateCoupon(ctx context.Context, code string, cartTotal float64) (*CouponValidationResult, error)
	ListCoupons(ctx context.Context, limit, offset int) ([]Coupon, error)
	CountCoupons(ctx context.Context) (int64, error)
	UpdateCoupon(ctx context.Context, coupon *Coupon) error
	IncrementCouponUsage(ctx context.Context, id uuid.UUID) error
	DeleteCoupon(ctx context.Context, id uuid.UUID) error
}
