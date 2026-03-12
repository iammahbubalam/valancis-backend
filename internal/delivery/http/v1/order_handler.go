package v1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"valancis-backend/internal/domain"
	"valancis-backend/internal/usecase"
)

type OrderHandler struct {
	orderUC         *usecase.OrderUsecase
	maxCartQuantity int
}

func NewOrderHandler(uc *usecase.OrderUsecase, maxCartQuantity int) *OrderHandler {
	return &OrderHandler{
		orderUC:         uc,
		maxCartQuantity: maxCartQuantity,
	}
}

// --- Cart Handlers ---

func (h *OrderHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	cart, err := h.orderUC.GetMyCart(r.Context(), user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}

type addToCartReq struct {
	ProductID string  `json:"productId"`
	VariantID *string `json:"variantId"`
	Quantity  int     `json:"quantity"`
}

func (h *OrderHandler) AddToCart(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req addToCartReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	slog.Info("Handler: AddToCart Request", "user_id", user.ID, "product_id", req.ProductID, "variant_id", *req.VariantID, "quantity", req.Quantity)

	// L9: Validate quantity bounds
	if req.Quantity <= 0 {
		http.Error(w, "Quantity must be positive", http.StatusBadRequest)
		return
	}
	if req.Quantity > h.maxCartQuantity {
		http.Error(w, "Quantity exceeds maximum limit", http.StatusBadRequest)
		return
	}

	cart, err := h.orderUC.AddToCart(r.Context(), user.ID, req.ProductID, req.VariantID, req.Quantity)
	if err != nil {
		slog.Error("AddToCart failed", "user_id", user.ID, "product_id", req.ProductID, "error", err)

		statusCode := http.StatusInternalServerError
		errMsg := err.Error()
		if strings.Contains(errMsg, "insufficient stock") || strings.Contains(errMsg, "out of stock") || strings.Contains(errMsg, "product unavailable") || strings.Contains(errMsg, "not found") {
			statusCode = http.StatusBadRequest
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{
			"error": errMsg,
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}

func (h *OrderHandler) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	productID := r.PathValue("productId")
	if productID == "" {
		http.Error(w, "Product ID required", http.StatusBadRequest)
		return
	}
	variantID := r.URL.Query().Get("variantId")
	if variantID == "" {
		http.Error(w, "Variant ID required", http.StatusBadRequest)
		return
	}

	cart, err := h.orderUC.RemoveFromCart(r.Context(), user.ID, productID, variantID)
	if err != nil {
		slog.Error("RemoveFromCart failed", "user_id", user.ID, "product_id", productID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}

func (h *OrderHandler) UpdateCart(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ProductID string  `json:"productId"`
		VariantID *string `json:"variantId"`
		Quantity  int     `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	slog.Info("Handler: UpdateCart Request", "user_id", user.ID, "product_id", req.ProductID, "variant_id", *req.VariantID, "quantity", req.Quantity)

	cart, err := h.orderUC.UpdateCartItemQuantity(r.Context(), user.ID, req.ProductID, req.VariantID, req.Quantity)
	if err != nil {
		slog.Error("UpdateCart failed", "user_id", user.ID, "product_id", req.ProductID, "error", err)

		statusCode := http.StatusInternalServerError
		errMsg := err.Error()
		if strings.Contains(errMsg, "insufficient stock") || strings.Contains(errMsg, "out of stock") || strings.Contains(errMsg, "not found") {
			statusCode = http.StatusBadRequest
		}

		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{
			"message": errMsg,
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}

// --- Order Handlers ---

func (h *OrderHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req usecase.CheckoutReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// L9: Extract Identity Markers for high CAPI IMQ
	req.IPAddress = r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		req.IPAddress = strings.Split(forwarded, ",")[0]
	}
	req.UserAgent = r.Header.Get("User-Agent")

	// Extract Facebook Cookies: prefer JSON body (from frontend), fallback to HTTP cookies
	if req.FBP == "" {
		if fbp, err := r.Cookie("_fbp"); err == nil {
			req.FBP = fbp.Value
		}
	}
	if req.FBC == "" {
		if fbc, err := r.Cookie("_fbc"); err == nil {
			req.FBC = fbc.Value
		}
	}

	order, err := h.orderUC.Checkout(r.Context(), user.ID, req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		errMsg := err.Error()
		statusCode := http.StatusInternalServerError

		if strings.Contains(errMsg, "insufficient stock") || strings.Contains(errMsg, "out of stock") || strings.Contains(errMsg, "cart is empty") || strings.Contains(errMsg, "not found") {
			statusCode = http.StatusBadRequest
		}

		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{
			"message": errMsg,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *OrderHandler) GetMyOrders(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	orders, err := h.orderUC.GetMyOrders(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

type ApplyCouponReq struct {
	CouponCode string `json:"couponCode"`
}

func (h *OrderHandler) ApplyCoupon(w http.ResponseWriter, r *http.Request) {
	// Coupon system is deactivated
	w.WriteHeader(http.StatusGone)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Coupon system is currently deactivated",
		"valid":   "false",
	})
}
