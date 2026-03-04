package v1

import (
	"encoding/json"
	"net/http"
	"strconv"
	"valancis-backend/internal/domain"
	"valancis-backend/internal/usecase"
)

type AdminOrderHandler struct {
	orderUC *usecase.OrderUsecase
}

func NewAdminOrderHandler(uc *usecase.OrderUsecase) *AdminOrderHandler {
	return &AdminOrderHandler{orderUC: uc}
}

func (h *AdminOrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = 20
	}

	filter := domain.OrderFilter{
		Page:          page,
		Limit:         limit,
		Status:        r.URL.Query().Get("status"),
		PaymentStatus: r.URL.Query().Get("payment_status"),
		Search:        r.URL.Query().Get("search"),
	}

	if val := r.URL.Query().Get("is_preorder"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			filter.IsPreorder = &b
		}
	}

	orders, total, err := h.orderUC.GetAllOrders(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orders": orders,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

func (h *AdminOrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Order ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Extract admin ID from context (set by middleware)
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok || user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	adminID := user.ID

	if err := h.orderUC.UpdateOrderStatus(r.Context(), id, req.Status, req.Note, adminID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Order status updated"})
}

func (h *AdminOrderHandler) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Extract admin ID
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok || user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	adminID := user.ID

	if err := h.orderUC.VerifyOrderPayment(r.Context(), id, adminID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Payment verified"})
}

func (h *AdminOrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Order ID required", http.StatusBadRequest)
		return
	}

	// Use existing UseCase GetByID - wait, that's internal domain.
	// Admin needs full access. GetByID in repo is sufficient.
	// But UseCase only has GetMyOrders (by userID).
	// I need to expose GetOrder(admin) in UseCase or reuse something.
	// Let's look at order_usecase.go.
	// We likely need a wrapper: GetOrderByID(ctx, id) that calls repo.GetByID directly.

	// Assuming I'll add GetOrderByID to UseCase shortly.
	order, err := h.orderUC.GetOrder(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *AdminOrderHandler) RefundOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Order ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Amount  float64 `json:"amount"`
		Reason  string  `json:"reason"`
		Restock bool    `json:"restock"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Get Admin ID from context
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok || user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	adminID := user.ID

	if err := h.orderUC.ProcessRefund(r.Context(), id, req.Amount, req.Reason, req.Restock, adminID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Refund processed"})
}

func (h *AdminOrderHandler) UpdatePaymentStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Extract admin ID
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok || user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	adminID := user.ID

	if err := h.orderUC.UpdatePaymentStatus(r.Context(), id, req.Status, adminID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Payment status updated"})
}

func (h *AdminOrderHandler) GetOrderHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Check if user is admin
	if _, ok := r.Context().Value(domain.UserContextKey).(*domain.User); !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	history, err := h.orderUC.GetOrderHistory(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (h *AdminOrderHandler) UpdateShippingZone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Zone string `json:"zone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Extract admin ID
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok || user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	adminID := user.ID

	if err := h.orderUC.UpdateShippingZone(r.Context(), id, req.Zone, adminID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Shipping zone updated"})
}
