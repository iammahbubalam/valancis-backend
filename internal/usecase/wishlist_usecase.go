package usecase

import (
	"context"
	"valancis-backend/internal/domain"
)

type WishlistUsecase struct {
	repo domain.WishlistRepository
}

func NewWishlistUsecase(repo domain.WishlistRepository) *WishlistUsecase {
	return &WishlistUsecase{
		repo: repo,
	}
}

func (u *WishlistUsecase) GetMyWishlist(ctx context.Context, userID string) (*domain.Wishlist, error) {
	// 1. Try to get existing wishlist
	wishlist, err := u.repo.GetWishlistByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 2. If not found, create one
	if wishlist == nil {
		wishlist, err = u.repo.CreateWishlist(ctx, userID)
		if err != nil {
			return nil, err
		}
	}

	// 3. Fetch items
	items, err := u.repo.GetWishlistItems(ctx, wishlist.ID)
	if err != nil {
		return nil, err
	}
	wishlist.Items = items

	return wishlist, nil
}

func (u *WishlistUsecase) AddToWishlist(ctx context.Context, userID, productID string) error {
	wishlist, err := u.GetMyWishlist(ctx, userID)
	if err != nil {
		return err
	}
	return u.repo.AddWishlistItem(ctx, wishlist.ID, productID)
}

func (u *WishlistUsecase) RemoveFromWishlist(ctx context.Context, userID, productID string) error {
	wishlist, err := u.GetMyWishlist(ctx, userID) // Could optimize to just get ID
	if err != nil {
		return err
	}
	return u.repo.RemoveWishlistItem(ctx, wishlist.ID, productID)
}

func (u *WishlistUsecase) IsInWishlist(ctx context.Context, userID, productID string) (bool, error) {
	wishlist, err := u.GetMyWishlist(ctx, userID)
	if err != nil {
		return false, err
	}
	return u.repo.CheckItemInWishlist(ctx, wishlist.ID, productID)
}
