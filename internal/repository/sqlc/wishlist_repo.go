package sqlcrepo

import (
	"context"
	"encoding/json"
	"valancis-backend/db/sqlc"
	"valancis-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type wishlistRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewWishlistRepository(db *pgxpool.Pool) domain.WishlistRepository {
	return &wishlistRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *wishlistRepository) GetWishlistByUserID(ctx context.Context, userID string) (*domain.Wishlist, error) {
	row, err := r.queries.GetWishlistByUserID(ctx, stringToUUID(userID))
	if err != nil {
		if err.Error() == "no rows in result set" || err.Error() == "sql: no rows in result set" {
			return nil, nil // Return nil if not found
		}
		return nil, err
	}

	return &domain.Wishlist{
		ID:        uuidToString(row.ID),
		UserID:    uuidToString(row.UserID),
		CreatedAt: pgtimeToTime(row.CreatedAt),
		UpdatedAt: pgtimeToTime(row.UpdatedAt),
	}, nil
}

func (r *wishlistRepository) CreateWishlist(ctx context.Context, userID string) (*domain.Wishlist, error) {
	row, err := r.queries.CreateWishlist(ctx, stringToUUID(userID))
	if err != nil {
		return nil, err
	}
	return &domain.Wishlist{
		ID:        uuidToString(row.ID),
		UserID:    uuidToString(row.UserID),
		CreatedAt: pgtimeToTime(row.CreatedAt),
		UpdatedAt: pgtimeToTime(row.UpdatedAt),
	}, nil
}

func (r *wishlistRepository) GetWishlistItems(ctx context.Context, wishlistID string) ([]domain.WishlistItem, error) {
	rows, err := r.queries.GetWishlistItems(ctx, stringToUUID(wishlistID))
	if err != nil {
		return nil, err
	}

	items := make([]domain.WishlistItem, 0, len(rows))
	for _, row := range rows {
		item := domain.WishlistItem{
			ID:        uuidToString(row.WishlistItemID),
			ProductID: uuidToString(row.ProductID),
			AddedAt:   pgtimeToTime(row.AddedAt),
			Product: domain.Product{
				ID:        uuidToString(row.ProductID),
				Name:      row.Name,
				Slug:      row.Slug,
				BasePrice: numericToFloat64(row.BasePrice),
				SalePrice: numericToFloat64Ptr(row.SalePrice),
				Stock:     int(row.TotalStock),
			},
		}

		// Sync StockStatus
		if item.Product.Stock <= 0 {
			item.Product.StockStatus = "out_of_stock"
		} else {
			item.Product.StockStatus = "in_stock"
		}

		// TotalStock available in row.TotalStock if needed for UI, but domain.Product doesn't hold it.
		// We could add it to WishlistItem if required.

		if len(row.Media) > 0 {
			item.Product.Media = domain.RawJSON(row.Media)
			var images []string
			if err := json.Unmarshal(row.Media, &images); err == nil {
				item.Product.Images = images
			}
		}

		items = append(items, item)
	}
	return items, nil
}

func (r *wishlistRepository) AddWishlistItem(ctx context.Context, wishlistID, productID string) error {
	return r.queries.AddWishlistItem(ctx, sqlc.AddWishlistItemParams{
		WishlistID: stringToUUID(wishlistID),
		ProductID:  stringToUUID(productID),
	})
}

func (r *wishlistRepository) RemoveWishlistItem(ctx context.Context, wishlistID, productID string) error {
	return r.queries.RemoveWishlistItem(ctx, sqlc.RemoveWishlistItemParams{
		WishlistID: stringToUUID(wishlistID),
		ProductID:  stringToUUID(productID),
	})
}

func (r *wishlistRepository) CheckItemInWishlist(ctx context.Context, wishlistID, productID string) (bool, error) {
	exists, err := r.queries.CheckItemInWishlist(ctx, sqlc.CheckItemInWishlistParams{
		WishlistID: stringToUUID(wishlistID),
		ProductID:  stringToUUID(productID),
	})
	if err != nil {
		return false, err
	}
	return exists, nil
}
