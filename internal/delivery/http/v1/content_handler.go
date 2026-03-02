package v1

import (
	"encoding/json"
	"net/http"
	"valancis-backend/internal/usecase"
)

type ContentHandler struct {
	usecase usecase.ContentUsecase
}

func NewContentHandler(u usecase.ContentUsecase) *ContentHandler {
	return &ContentHandler{usecase: u}
}

func (h *ContentHandler) GetContent(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	content, err := h.usecase.GetContent(r.Context(), key)
	if err != nil {
		// If not found, return an empty object or 404.
		// For dynamic config, returning 404 is appropriate.
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(content)
}

func (h *ContentHandler) UpsertContent(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	var body interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	updated, err := h.usecase.UpsertContent(r.Context(), key, body)
	if err != nil {
		http.Error(w, "Failed to upsert content", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}
