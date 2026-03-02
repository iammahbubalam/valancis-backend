package v1

import (
	"encoding/json"
	"net/http"
	"strconv"

	"valancis-backend/internal/domain"
)

type SearchHandler struct {
	searchUC domain.SearchUsecase
}

func NewSearchHandler(searchUC domain.SearchUsecase) *SearchHandler {
	return &SearchHandler{
		searchUC: searchUC,
	}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	if query == "" {
		response := domain.Response{
			Success: true,
			Data:    []domain.Product{},
			Meta: &domain.Pagination{
				Page:       page,
				Limit:      limit,
				TotalItems: 0,
				TotalPages: 0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	products, pagination, err := h.searchUC.Search(r.Context(), query, page, limit)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	response := domain.Response{
		Success: true,
		Data:    products,
		Meta:    &pagination,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
