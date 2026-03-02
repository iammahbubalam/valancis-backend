package v1

import (
	"encoding/json"
	"net/http"
	"valancis-backend/internal/domain"
	"valancis-backend/pkg/cache"
	"strconv"
)

type AdminConfigHandler struct {
	cache      cache.CacheService
	configRepo domain.ConfigRepository
}

func NewAdminConfigHandler(cache cache.CacheService, configRepo domain.ConfigRepository) *AdminConfigHandler {
	return &AdminConfigHandler{cache: cache, configRepo: configRepo}
}

func (h *AdminConfigHandler) UpdateShippingZone(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var req domain.ShippingZone
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.ID = int32(id)

	updated, err := h.configRepo.UpdateShippingZone(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.cache.Delete("system:config:enums")

	json.NewEncoder(w).Encode(updated)
}

func (h *AdminConfigHandler) GetAllShippingZones(w http.ResponseWriter, r *http.Request) {
	zones, err := h.configRepo.GetAllShippingZones(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(zones)
}

func (h *AdminConfigHandler) CreateShippingZone(w http.ResponseWriter, r *http.Request) {
	var req domain.ShippingZone
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	created, err := h.configRepo.CreateShippingZone(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.cache.Delete("system:config:enums")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *AdminConfigHandler) DeleteShippingZone(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.configRepo.DeleteShippingZone(r.Context(), int32(id)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.cache.Delete("system:config:enums")

	w.WriteHeader(http.StatusNoContent)
}
