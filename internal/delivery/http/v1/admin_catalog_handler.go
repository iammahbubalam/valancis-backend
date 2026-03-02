package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"valancis-backend/internal/domain"
	"valancis-backend/internal/usecase"
	"valancis-backend/pkg/utils"
	"strconv"

	"github.com/google/uuid"
)

type AdminCatalogHandler struct {
	catalogUC *usecase.CatalogUsecase
}

func NewAdminCatalogHandler(uc *usecase.CatalogUsecase) *AdminCatalogHandler {
	return &AdminCatalogHandler{catalogUC: uc}
}

// GetAllCategories returns a FLAT list of categories (no hierarchy) for admin selection dropdowns
// Query params: isActive (optional) - "true", "false", or omit for all
func (h *AdminCatalogHandler) GetAllCategories(w http.ResponseWriter, r *http.Request) {
	// Parse optional isActive filter
	var isActive *bool
	if val := r.URL.Query().Get("isActive"); val != "" {
		switch val {
		case "true":
			t := true
			isActive = &t
		case "false":
			f := false
			isActive = &f
		}
	}

	cats, err := h.catalogUC.GetCategoriesFlat(r.Context(), isActive)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cats)
}

func (h *AdminCatalogHandler) GetProductStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.catalogUC.GetProductStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

type productReq struct {
	domain.Product
	CategoryIDs   []string `json:"categoryIds"`
	CollectionIDs []string `json:"collectionIds"`
}

func (h *AdminCatalogHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req productReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	product := req.Product

	// L9: Normalize optional price fields (0 → nil for DB constraint)
	if product.SalePrice != nil && *product.SalePrice <= 0 {
		product.SalePrice = nil
	}

	// Map CategoryIDs to Categories
	if len(req.CategoryIDs) > 0 {
		product.Categories = make([]domain.Category, len(req.CategoryIDs))
		for i, id := range req.CategoryIDs {
			product.Categories[i] = domain.Category{ID: id}
		}
	}

	// Map CollectionIDs to Collections
	if len(req.CollectionIDs) > 0 {
		product.Collections = make([]domain.Collection, len(req.CollectionIDs))
		for i, id := range req.CollectionIDs {
			product.Collections[i] = domain.Collection{ID: id}
		}
	}

	if err := h.catalogUC.CreateProduct(r.Context(), &product); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

func (h *AdminCatalogHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	filter := domain.ProductFilter{
		Limit:  20,
		Offset: 0,
		Sort:   "created_at desc",
	}

	// Pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			filter.Limit = l
		}
	}
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			filter.Offset = (p - 1) * filter.Limit
		}
	}

	// Sort
	if sort := r.URL.Query().Get("sort"); sort != "" {
		filter.Sort = sort
	}

	// Filter Search
	if q := r.URL.Query().Get("search"); q != "" {
		filter.Query = q
	}

	// Filter Category
	if cat := r.URL.Query().Get("category"); cat != "" {
		filter.CategorySlug = cat
	}

	// Filter Status (isActive)
	// Expect "true", "false", "all" (or empty)
	status := r.URL.Query().Get("isActive")
	switch status {
	case "true":
		t := true
		filter.IsActive = &t
	case "false":
		f := false
		filter.IsActive = &f
	default:
		// Explicitly nil for "all" or empty
		filter.IsActive = nil
	}

	products, total, err := h.catalogUC.ListProducts(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  products,
		"total": total,
		"page":  (filter.Offset / filter.Limit) + 1,
		"limit": filter.Limit,
	})
}

func (h *AdminCatalogHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Product ID required", http.StatusBadRequest)
		return
	}

	var req productReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	product := req.Product
	product.ID = id

	// L9: Normalize optional price fields (0 → nil for DB constraint)
	if product.SalePrice != nil && *product.SalePrice <= 0 {
		product.SalePrice = nil
	}

	// Map CategoryIDs to Categories
	if len(req.CategoryIDs) > 0 {
		product.Categories = make([]domain.Category, len(req.CategoryIDs))
		for i, id := range req.CategoryIDs {
			product.Categories[i] = domain.Category{ID: id}
		}
	} else if req.CategoryIDs != nil {
		// Explicit empty array clears categories
		product.Categories = []domain.Category{}
	}

	// Map CollectionIDs to Collections
	if len(req.CollectionIDs) > 0 {
		product.Collections = make([]domain.Collection, len(req.CollectionIDs))
		for i, id := range req.CollectionIDs {
			product.Collections[i] = domain.Collection{ID: id}
		}
	} else if req.CollectionIDs != nil {
		// Explicit empty array clears collections
		product.Collections = []domain.Collection{}
	}

	if err := h.catalogUC.UpdateProduct(r.Context(), &product); err != nil {
		fmt.Printf("ERROR UpdateProduct: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *AdminCatalogHandler) UpdateProductStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Product ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		IsActive bool `json:"isActive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := h.catalogUC.UpdateProductStatus(r.Context(), id, req.IsActive); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// L9 Pattern: Return the Delta State so frontend can update local cache accurately
	// without refetching the entire list or stats.
	response := map[string]interface{}{
		"id":       id,
		"isActive": req.IsActive,
		"status":   "updated",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *AdminCatalogHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Product ID required", http.StatusBadRequest)
		return
	}

	if err := h.catalogUC.DeleteProduct(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (h *AdminCatalogHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	idOrSlug := r.PathValue("id")
	if idOrSlug == "" {
		http.Error(w, "Product ID or Slug required", http.StatusBadRequest)
		return
	}

	var product *domain.Product
	var err error

	// Check if valid UUID
	if _, uuidErr := uuid.Parse(idOrSlug); uuidErr == nil {
		product, err = h.catalogUC.GetProductByID(r.Context(), idOrSlug)
	} else {
		// Assume Slug
		product, err = h.catalogUC.GetProductDetails(r.Context(), idOrSlug)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(product)
}

func (h *AdminCatalogHandler) GetVariantList(w http.ResponseWriter, r *http.Request) {
	filter := domain.VariantListFilter{
		ProductID:    r.URL.Query().Get("productId"),
		LowStockOnly: r.URL.Query().Get("lowStockOnly") == "true",
		Search:       r.URL.Query().Get("search"),
		Sort:         r.URL.Query().Get("sort"),
		Limit:        utils.ParseInt(r.URL.Query().Get("limit"), 50),
		Offset:       utils.ParseInt(r.URL.Query().Get("offset"), 0),
	}

	variants, pagination, err := h.catalogUC.GetVariantList(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := domain.Response{
		Success: true,
		Data:    variants,
		Meta:    pagination,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

type adjustStockReq struct {
	VariantID    string `json:"variantId"`
	ChangeAmount int    `json:"changeAmount"` // negative to deduct
	Reason       string `json:"reason"`
}

func (h *AdminCatalogHandler) AdjustStock(w http.ResponseWriter, r *http.Request) {
	// Require Admin (Handled by middleware)
	// Get ID from authenticated user for reference (optional)
	// For now, we just take the body.

	var req adjustStockReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Get Admin ID from context
	adminUser, _ := r.Context().Value(domain.UserContextKey).(*domain.User)
	referenceID := "admin"
	if adminUser != nil {
		referenceID = adminUser.ID
	}

	if err := h.catalogUC.AdjustStock(r.Context(), req.VariantID, req.ChangeAmount, req.Reason, referenceID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "stock updated"})
}

func (h *AdminCatalogHandler) GetInventoryLogs(w http.ResponseWriter, r *http.Request) {
	productID := r.URL.Query().Get("productId")

	limit := 20
	offset := 0
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		offset = (p - 1) * limit
	}

	logs, total, err := h.catalogUC.GetInventoryLogs(r.Context(), productID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  logs,
		"total": total,
		"page":  (offset / limit) + 1,
		"limit": limit,
	})
}

func (h *AdminCatalogHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var category domain.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := h.catalogUC.CreateCategory(r.Context(), &category); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(category)
}

func (h *AdminCatalogHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Category ID required", http.StatusBadRequest)
		return
	}

	var category domain.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	category.ID = id

	if err := h.catalogUC.UpdateCategory(r.Context(), &category); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *AdminCatalogHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Category ID required", http.StatusBadRequest)
		return
	}

	if err := h.catalogUC.DeleteCategory(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

type reorderReq struct {
	Updates []domain.CategoryReorderItem `json:"updates"`
}

func (h *AdminCatalogHandler) ReorderCategories(w http.ResponseWriter, r *http.Request) {
	var req reorderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := h.catalogUC.ReorderCategories(r.Context(), req.Updates); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "reordered"})
}

// --- Collections ---

func (h *AdminCatalogHandler) GetAllCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := h.catalogUC.GetAllCollections(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collections)
}

func (h *AdminCatalogHandler) CreateCollection(w http.ResponseWriter, r *http.Request) {
	var collection domain.Collection
	if err := json.NewDecoder(r.Body).Decode(&collection); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	fmt.Printf("DEBUG: CreateCollection Payload: %+v, IsActive: %v\n", collection, collection.IsActive)

	if err := h.catalogUC.CreateCollection(r.Context(), &collection); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(collection)
}

func (h *AdminCatalogHandler) UpdateCollection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	var collection domain.Collection
	if err := json.NewDecoder(r.Body).Decode(&collection); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	collection.ID = id

	if err := h.catalogUC.UpdateCollection(r.Context(), &collection); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *AdminCatalogHandler) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	if err := h.catalogUC.DeleteCollection(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (h *AdminCatalogHandler) ManageCollectionProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // Collection ID
	var req struct {
		ProductID string `json:"productId"`
		Action    string `json:"action"` // "add" or "remove"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	var err error
	if req.Action == "add" {
		err = h.catalogUC.AddProductToCollection(r.Context(), id, req.ProductID)
	} else {
		err = h.catalogUC.RemoveProductFromCollection(r.Context(), id, req.ProductID)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
