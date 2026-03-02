package v1

import (
	"net/http"
	"valancis-backend/internal/usecase"
)

// AdminCouponHandler handles admin coupon management endpoints.
// L9: Thin handler layer - delegates all logic to usecase.
type AdminCouponHandler struct {
	couponUC *usecase.CouponUsecase
}

// NewAdminCouponHandler creates a new AdminCouponHandler.
func NewAdminCouponHandler(uc *usecase.CouponUsecase) *AdminCouponHandler {
	return &AdminCouponHandler{couponUC: uc}
}

// ListCoupons returns paginated list of all coupons.
// GET /api/v1/admin/coupons?page=1&limit=20
func (h *AdminCouponHandler) ListCoupons(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Coupon system is currently deactivated", http.StatusGone)
}

// CreateCoupon creates a new coupon.
// POST /api/v1/admin/coupons
func (h *AdminCouponHandler) CreateCoupon(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Coupon system is currently deactivated", http.StatusGone)
}

// GetCoupon returns a single coupon by ID.
// GET /api/v1/admin/coupons/{id}
func (h *AdminCouponHandler) GetCoupon(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Coupon system is currently deactivated", http.StatusGone)
}

// UpdateCoupon updates an existing coupon.
// PUT /api/v1/admin/coupons/{id}
func (h *AdminCouponHandler) UpdateCoupon(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Coupon system is currently deactivated", http.StatusGone)
}

// DeleteCoupon deletes a coupon by ID.
// DELETE /api/v1/admin/coupons/{id}
func (h *AdminCouponHandler) DeleteCoupon(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Coupon system is currently deactivated", http.StatusGone)
}

// isValidationError checks if an error is a validation error based on message.
// L9: Simple heuristic - production would use typed errors.
func isValidationError(err error) bool {
	msg := err.Error()
	validationPhrases := []string{
		"is required",
		"must be",
		"cannot exceed",
		"already exists",
		"not found",
		"invalid",
	}
	for _, phrase := range validationPhrases {
		if stringContains(msg, phrase) {
			return true
		}
	}
	return false
}

// stringContains is a simple substring check helper.
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
