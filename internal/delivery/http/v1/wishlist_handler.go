package v1

import (
	"encoding/json"
	"net/http"
	"valancis-backend/internal/domain"
	"valancis-backend/internal/usecase"
	"valancis-backend/pkg/utils"
)

type WishlistHandler struct {
	usecase *usecase.WishlistUsecase
}

func NewWishlistHandler(usecase *usecase.WishlistUsecase) *WishlistHandler {
	return &WishlistHandler{usecase: usecase}
}

func (h *WishlistHandler) GetMyWishlist(w http.ResponseWriter, r *http.Request) {
	// Require Auth Middleware to set user_id
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	wishlist, err := h.usecase.GetMyWishlist(r.Context(), user.ID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, wishlist)
}

type WishlistRequest struct {
	ProductID string `json:"productId"`
}

func (h *WishlistHandler) AddToWishlist(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req WishlistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.usecase.AddToWishlist(r.Context(), user.ID, req.ProductID); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to add to wishlist")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "Added to wishlist"})
}

func (h *WishlistHandler) RemoveFromWishlist(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	productID := r.PathValue("productId")

	if err := h.usecase.RemoveFromWishlist(r.Context(), user.ID, productID); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to remove from wishlist")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "Removed from wishlist"})
}
