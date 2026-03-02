package v1

import (
	"encoding/json"
	"net/http"
	"valancis-backend/internal/domain"
	"valancis-backend/pkg/cache"
	"time"
)

type ConfigHandler struct {
	cache      cache.CacheService
	configRepo domain.ConfigRepository
}

func NewConfigHandler(cache cache.CacheService, configRepo domain.ConfigRepository) *ConfigHandler {
	return &ConfigHandler{cache: cache, configRepo: configRepo}
}

// GET /api/v1/config/enums
func (h *ConfigHandler) GetEnums(w http.ResponseWriter, r *http.Request) {
	// Cache Key
	cacheKey := "system:config:enums"

	// Check Cache
	if val, found := h.cache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		// Start Cache Headers
		w.Header().Set("Cache-Control", "public, max-age=3600")
		json.NewEncoder(w).Encode(val)
		return
	}

	zones, _ := h.configRepo.GetActiveShippingZones(r.Context())

	response := map[string]interface{}{
		"orderStatuses":   domain.OrderStatuses,
		"paymentStatuses": domain.PaymentStatuses,
		"paymentMethods":  domain.PaymentMethods,
		"shippingZones":   zones,
	}

	h.cache.Set(cacheKey, response, 1*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	json.NewEncoder(w).Encode(response)
}
