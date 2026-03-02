package v1

import (
	"encoding/json"
	"net/http"
	"valancis-backend/internal/domain"
	"valancis-backend/internal/usecase"
	"strconv"
)

type CatalogHandler struct {
	catalogUC *usecase.CatalogUsecase
}

func NewCatalogHandler(uc *usecase.CatalogUsecase) *CatalogHandler {
	return &CatalogHandler{catalogUC: uc}
}

func (h *CatalogHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.catalogUC.GetCategoryTree(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cats)
}

func (h *CatalogHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit == 0 {
		limit = 20
	}
	page, _ := strconv.Atoi(query.Get("page"))
	if page == 0 {
		page = 1
	}

	minPrice, _ := strconv.ParseFloat(query.Get("min_price"), 64)
	maxPrice, _ := strconv.ParseFloat(query.Get("max_price"), 64)

	var isFeatured *bool
	if val := query.Get("is_featured"); val != "" {
		b, err := strconv.ParseBool(val)
		if err == nil {
			isFeatured = &b
		}
	}

	filter := domain.ProductFilter{
		CategorySlug: query.Get("category_slug"),
		Query:        query.Get("q"),
		Sort:         query.Get("sort"),
		MinPrice:     minPrice,
		MaxPrice:     maxPrice,
		Limit:        limit,
		Offset:       (page - 1) * limit,
		IsFeatured:   isFeatured,
	}

	products, total, err := h.catalogUC.ListProducts(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": products,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

func (h *CatalogHandler) GetProductDetails(w http.ResponseWriter, r *http.Request) {
	// Simple Slug extraction - in standard mux with Go 1.22 we can use PathValue
	// But let's assume standard behavior: /products/{slug}
	// Note: We need to register this correctly in mux

	slug := r.PathValue("slug")
	if slug == "" {
		http.Error(w, "Slug required", http.StatusBadRequest)
		return
	}

	product, err := h.catalogUC.GetProductDetails(r.Context(), slug)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

func (h *CatalogHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Product ID required", http.StatusBadRequest)
		return
	}

	product, err := h.catalogUC.GetProductByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

func (h *CatalogHandler) AddReview(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	productID := r.PathValue("id")
	if productID == "" {
		http.Error(w, "Product ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// L9: Validate rating bounds
	if req.Rating < 1 || req.Rating > 5 {
		http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	if err := h.catalogUC.AddReview(r.Context(), user.ID, productID, req.Rating, req.Comment); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "review added"})
}

func (h *CatalogHandler) GetReviews(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	if productID == "" {
		http.Error(w, "Product ID required", http.StatusBadRequest)
		return
	}

	reviews, err := h.catalogUC.GetProductReviews(r.Context(), productID)
	if err != nil {
		http.Error(w, "Failed to fetch reviews", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}

// --- Collections ---

func (h *CatalogHandler) GetCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := h.catalogUC.GetCollections(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collections)
}

func (h *CatalogHandler) GetAllCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := h.catalogUC.GetAllCollections(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collections)
}

func (h *CatalogHandler) GetCollectionBySlug(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		http.Error(w, "Slug required", http.StatusBadRequest)
		return
	}
	collection, err := h.catalogUC.GetCollectionBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collection)
}
