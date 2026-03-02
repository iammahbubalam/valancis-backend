package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"valancis-backend/internal/usecase"
	"strconv"
	"time"
)

// L9 Admin Stats Handler: Zero hardcoded values, all params from query string
type AdminStatsHandler struct {
	statsUC *usecase.StatsUsecase
}

func NewAdminStatsHandler(uc *usecase.StatsUsecase) *AdminStatsHandler {
	return &AdminStatsHandler{statsUC: uc}
}

// Helper: Parse required date param
func parseRequiredDate(r *http.Request, param string) (time.Time, error) {
	str := r.URL.Query().Get(param)
	if str == "" {
		return time.Time{}, fmt.Errorf("%s parameter is required", param)
	}
	return time.Parse("2006-01-02", str)
}

// Helper: Parse optional int32 param with default
func parseInt32WithDefault(r *http.Request, param string, defaultVal int32) int32 {
	str := r.URL.Query().Get(param)
	if str == "" {
		return defaultVal
	}
	val, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return defaultVal
	}
	return int32(val)
}

// GET /admin/stats/revenue?start=2024-01-01&end=2024-01-31&limit=30&offset=0
func (h *AdminStatsHandler) GetDailySales(w http.ResponseWriter, r *http.Request) {
	start, err := parseRequiredDate(r, "start")
	if err != nil {
		http.Error(w, "start date required (format: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	end, err := parseRequiredDate(r, "end")
	if err != nil {
		http.Error(w, "end date required (format: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	limit := parseInt32WithDefault(r, "limit", 30)
	offset := parseInt32WithDefault(r, "offset", 0)

	sales, err := h.statsUC.GetDailySales(r.Context(), start, end, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sales)
}

// GET /admin/stats/kpis?start=2024-01-01&end=2024-01-31
func (h *AdminStatsHandler) GetRevenueKPIs(w http.ResponseWriter, r *http.Request) {
	start, err := parseRequiredDate(r, "start")
	if err != nil {
		http.Error(w, "start date required (format: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	end, err := parseRequiredDate(r, "end")
	if err != nil {
		http.Error(w, "end date required (format: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	kpis, err := h.statsUC.GetRevenueKPIs(r.Context(), start, end)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(kpis)
}

// GET /admin/stats/inventory/low-stock?threshold=10&limit=50
func (h *AdminStatsHandler) GetLowStockProducts(w http.ResponseWriter, r *http.Request) {
	threshold := parseInt32WithDefault(r, "threshold", 5)
	limit := parseInt32WithDefault(r, "limit", 50)

	products, err := h.statsUC.GetLowStockProducts(r.Context(), threshold, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

// GET /admin/stats/inventory/dead-stock?days=90&limit=50
func (h *AdminStatsHandler) GetDeadStockProducts(w http.ResponseWriter, r *http.Request) {
	days := parseInt32WithDefault(r, "days", 90)
	limit := parseInt32WithDefault(r, "limit", 50)

	products, err := h.statsUC.GetDeadStockProducts(r.Context(), days, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

// GET /admin/stats/products/top-selling?start=2024-01-01&end=2024-01-31&limit=25
func (h *AdminStatsHandler) GetTopSellingProducts(w http.ResponseWriter, r *http.Request) {
	start, err := parseRequiredDate(r, "start")
	if err != nil {
		http.Error(w, "start date required (format: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	end, err := parseRequiredDate(r, "end")
	if err != nil {
		http.Error(w, "end date required (format: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	limit := parseInt32WithDefault(r, "limit", 10)

	// Log start of request
	fmt.Printf("GetTopSellingProductsHandler: Start - start=%v, end=%v, limit=%d\n", start, end, limit)

	products, err := h.statsUC.GetTopSellingProducts(r.Context(), start, end, limit)
	if err != nil {
		// Log error
		fmt.Printf("GetTopSellingProductsHandler: Error - %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Log success
	fmt.Printf("GetTopSellingProductsHandler: Success - Found %d products\n", len(products))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

// GET /admin/stats/customers/top?start=2024-01-01&end=2024-01-31&limit=25
func (h *AdminStatsHandler) GetTopCustomers(w http.ResponseWriter, r *http.Request) {
	start, err := parseRequiredDate(r, "start")
	if err != nil {
		http.Error(w, "start date required (format: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	end, err := parseRequiredDate(r, "end")
	if err != nil {
		http.Error(w, "end date required (format: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	limit := parseInt32WithDefault(r, "limit", 10)

	customers, err := h.statsUC.GetCustomerLTV(r.Context(), start, end, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customers)
}

// GET /admin/stats/customers/retention?start=2024-01-01&end=2024-01-31
func (h *AdminStatsHandler) GetCustomerRetention(w http.ResponseWriter, r *http.Request) {
	start, err := parseRequiredDate(r, "start")
	if err != nil {
		http.Error(w, "start date required (format: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	end, err := parseRequiredDate(r, "end")
	if err != nil {
		http.Error(w, "end date required (format: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	retention, err := h.statsUC.GetCustomerRetention(r.Context(), start, end)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(retention)
}
