package domain

import (
	"context"
	"time"
)

type Wishlist struct {
	ID        string         `json:"id"`
	UserID    string         `json:"userId"`
	Items     []WishlistItem `json:"items"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type WishlistItem struct {
	ID        string    `json:"id"`
	ProductID string    `json:"productId"`
	Product   Product   `json:"product"`
	AddedAt   time.Time `json:"addedAt"`
}

type WishlistRepository interface {
	GetWishlistByUserID(ctx context.Context, userID string) (*Wishlist, error)
	CreateWishlist(ctx context.Context, userID string) (*Wishlist, error)
	GetWishlistItems(ctx context.Context, wishlistID string) ([]WishlistItem, error)
	AddWishlistItem(ctx context.Context, wishlistID, productID string) error
	RemoveWishlistItem(ctx context.Context, wishlistID, productID string) error
	CheckItemInWishlist(ctx context.Context, wishlistID, productID string) (bool, error)
}
