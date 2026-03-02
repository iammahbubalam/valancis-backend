package usecase

import (
	"context"
	"fmt"
	"valancis-backend/config"
	"valancis-backend/internal/domain"
	"valancis-backend/pkg/cache"
	"valancis-backend/pkg/storage"
	"valancis-backend/pkg/utils"
	"time"
)

type CatalogUsecase struct {
	repo      domain.ProductRepository
	orderRepo domain.OrderRepository
	cache     cache.CacheService
	storage   *storage.R2Storage
	cfg       *config.Config
}

func NewCatalogUsecase(repo domain.ProductRepository, orderRepo domain.OrderRepository, cache cache.CacheService, storage *storage.R2Storage, cfg *config.Config) *CatalogUsecase {
	return &CatalogUsecase{
		repo:      repo,
		orderRepo: orderRepo,
		cache:     cache,
		storage:   storage,
		cfg:       cfg,
	}
}

func (uc *CatalogUsecase) CreateProduct(ctx context.Context, product *domain.Product) error {
	// 1. Generate Slug if missing
	if product.Slug == "" {
		product.Slug = utils.GenerateSlug(product.Name)
	}
	// 2. Set Defaults
	if len(product.Variants) == 0 {
		// Valid case: standard product, repo will create Master Variant
	}
	// 3. Validate Pricing (L9 Data Integrity)
	if product.SalePrice != nil && *product.SalePrice > 0 {
		if *product.SalePrice >= product.BasePrice {
			return fmt.Errorf("sale price must be less than base price")
		}
	}

	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()
	product.IsActive = true

	uc.invalidateStatsCache()
	return uc.repo.CreateProduct(ctx, product)
}

func (uc *CatalogUsecase) UpdateProduct(ctx context.Context, product *domain.Product) error {
	// L9: Validate Pricing (Data Integrity)
	if product.SalePrice != nil && *product.SalePrice > 0 {
		if *product.SalePrice >= product.BasePrice {
			return fmt.Errorf("sale price must be less than base price")
		}
	}

	product.UpdatedAt = time.Now()

	// L9: Detect and Delete Removed Images (Cleanup Orphans)
	// 1. Fetch existing product state
	existing, err := uc.repo.GetProductByID(ctx, product.ID)
	if err == nil && existing != nil {
		// 2. Identify removed images from Main Product
		newImages := make(map[string]bool)
		for _, img := range product.Images {
			newImages[img] = true
		}
		for _, oldImg := range existing.Images {
			if !newImages[oldImg] && oldImg != "" {
				// Old image is NOT in new list -> Delete it
				_ = uc.storage.DeleteFile(ctx, oldImg)
			}
		}

		// 3. Identify removed images from Variants
		// This is tricky because variants might be added/removed/updated.
		// Simplified strategy: Collect ALL current variant images, and compare against ALL new variant images.
		// NOTE: This assumes images are unique to variants or shared logic handles it.
		// For safety, we only delete if the image is NOT used in ANY new variant OR the main product.

		// Re-build "Keep List" with ALL new images (Product + All Variants)
		keepList := make(map[string]bool)
		for _, img := range product.Images {
			keepList[img] = true
		}
		for _, v := range product.Variants {
			for _, img := range v.Images {
				keepList[img] = true
			}
		}

		// Check Old Variant Images
		for _, v := range existing.Variants {
			for _, oldImg := range v.Images {
				if !keepList[oldImg] && oldImg != "" {
					_ = uc.storage.DeleteFile(ctx, oldImg)
				}
			}
		}
	}

	// Prevent slug update? Or allow re-generation? Let's allow simple update for now.
	// Invalidate cache
	uc.cache.Delete(fmt.Sprintf("product:slug:%s", product.Slug))
	uc.invalidateStatsCache()
	return uc.repo.UpdateProduct(ctx, product)
}

func (uc *CatalogUsecase) UpdateProductStatus(ctx context.Context, id string, isActive bool) error {
	uc.invalidateStatsCache()
	return uc.repo.UpdateProductStatus(ctx, id, isActive)
}

func (uc *CatalogUsecase) DeleteProduct(ctx context.Context, id string) error {
	// L9: Clean up R2 Images before deleting DB record
	product, err := uc.repo.GetProductByID(ctx, id)
	if err == nil && product != nil {
		// Delete All Product Images (Main + Gallery are consolidated in Images)
		for _, img := range product.Images {
			if img != "" {
				_ = uc.storage.DeleteFile(ctx, img)
			}
		}
		// Delete Variant Images
		for _, v := range product.Variants {
			for _, img := range v.Images {
				if img != "" {
					_ = uc.storage.DeleteFile(ctx, img)
				}
			}
		}
	}

	uc.invalidateStatsCache()
	return uc.repo.DeleteProduct(ctx, id)
}

func (uc *CatalogUsecase) AdjustStock(ctx context.Context, variantID string, changeAmount int, reason, referenceID string) error {
	// 1. Fetch current stock (L9 Robustness)
	variant, err := uc.repo.GetVariantByID(ctx, variantID)
	if err != nil {
		return fmt.Errorf("failed to fetch variant: %w", err)
	}
	if variant == nil {
		return fmt.Errorf("variant not found")
	}

	// 2. Validate Negative Stock
	// Allow negative stock only if explicitly allowed (e.g., pre_order).
	// Currently enforcing strict non-negative policy for standard flow.
	newStock := variant.Stock + changeAmount
	if newStock < 0 {
		return fmt.Errorf("insufficient stock: current %d, deducting %d", variant.Stock, -changeAmount)
	}

	uc.invalidateStatsCache()
	return uc.repo.UpdateStock(ctx, variantID, changeAmount, reason, referenceID)
}

func (uc *CatalogUsecase) GetInventoryLogs(ctx context.Context, productID string, limit, offset int) ([]domain.InventoryLog, int64, error) {
	return uc.repo.GetInventoryLogs(ctx, productID, limit, offset)
}

func (u *CatalogUsecase) GetCategoryTree(ctx context.Context) ([]domain.Category, error) {
	key := "category:tree:all"
	if val, found := u.cache.Get(key); found {
		return val.([]domain.Category), nil
	}

	tree, err := u.repo.GetCategoryTree(ctx)
	if err != nil {
		return nil, err
	}

	u.cache.Set(key, tree, u.cfg.CacheCategoryTTL)
	return tree, nil
}

func (u *CatalogUsecase) GetNavCategoryTree(ctx context.Context) ([]domain.Category, error) {
	key := "category:tree:nav"
	if val, found := u.cache.Get(key); found {
		return val.([]domain.Category), nil
	}

	tree, err := u.repo.GetNavCategoryTree(ctx)
	if err != nil {
		return nil, err
	}

	u.cache.Set(key, tree, u.cfg.CacheCategoryTTL)
	return tree, nil
}

// GetCategoriesFlat returns a flat list of categories without hierarchy
// isActive: nil = all, true = active only, false = inactive only
func (u *CatalogUsecase) GetCategoriesFlat(ctx context.Context, isActive *bool) ([]domain.Category, error) {
	// Build cache key based on filter
	var key string
	if isActive == nil {
		key = "category:flat:all"
	} else if *isActive {
		key = "category:flat:active"
	} else {
		key = "category:flat:inactive"
	}

	if val, found := u.cache.Get(key); found {
		return val.([]domain.Category), nil
	}

	cats, err := u.repo.GetCategoriesFlat(ctx, isActive)
	if err != nil {
		return nil, err
	}

	u.cache.Set(key, cats, u.cfg.CacheCategoryTTL)
	return cats, nil
}

func (uc *CatalogUsecase) CreateCategory(ctx context.Context, category *domain.Category) error {
	if category.Name == "" {
		return fmt.Errorf("category name is required")
	}
	if category.Slug == "" {
		category.Slug = utils.GenerateSlug(category.Name)
	}
	// Check if slug is already taken
	existing, _ := uc.repo.GetCategoryBySlug(ctx, category.Slug)
	if existing != nil {
		return fmt.Errorf("slug '%s' is already taken", category.Slug)
	}
	if category.ID == "" {
		category.ID = utils.GenerateUUID()
	}
	uc.invalidateCategoryCache()
	return uc.repo.CreateCategory(ctx, category)
}

func (uc *CatalogUsecase) UpdateCategory(ctx context.Context, category *domain.Category) error {
	if category.ID == "" {
		return fmt.Errorf("category ID required")
	}
	// Invalidate cache
	uc.invalidateCategoryCache()
	return uc.repo.UpdateCategory(ctx, category)
}

func (uc *CatalogUsecase) DeleteCategory(ctx context.Context, id string) error {
	uc.invalidateCategoryCache()
	return uc.repo.DeleteCategory(ctx, id)
}

func (uc *CatalogUsecase) ReorderCategories(ctx context.Context, updates []domain.CategoryReorderItem) error {
	if err := uc.repo.ReorderCategories(ctx, updates); err != nil {
		return err
	}
	// Warm the cache immediately with fresh data
	tree, err := uc.repo.GetCategoryTree(ctx)
	if err == nil {
		uc.cache.Set("category:tree:all", tree, uc.cfg.CacheCategoryTTL)
	}
	return nil
}

func (u *CatalogUsecase) ListProducts(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, int64, error) {
	// Add business logic here if needed (e.g., validate filters)
	return u.repo.GetProducts(ctx, filter)
}

func (u *CatalogUsecase) GetProductDetails(ctx context.Context, slug string) (*domain.Product, error) {
	key := fmt.Sprintf("product:slug:%s", slug)
	if val, found := u.cache.Get(key); found {
		return val.(*domain.Product), nil
	}

	product, err := u.repo.GetProductBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if product != nil {
		u.cache.Set(key, product, u.cfg.CacheProductTTL)
	}

	return product, nil
}

func (u *CatalogUsecase) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	return u.repo.GetProductByID(ctx, id)
}

func (u *CatalogUsecase) AddReview(ctx context.Context, userID, productID string, rating int, comment string) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	// VERIFICATION: Check if user purchased the product
	hasPurchased, err := u.orderRepo.HasPurchasedProduct(ctx, userID, productID)
	if err != nil {
		return fmt.Errorf("failed to verify purchase: %w", err)
	}
	if !hasPurchased {
		return fmt.Errorf("you can only review products you have purchased and received")
	}

	review := &domain.Review{
		ID:        utils.GenerateUUID(),
		UserID:    userID,
		ProductID: productID,
		Rating:    rating,
		Comment:   comment,
		CreatedAt: time.Now(),
	}

	return u.repo.CreateReview(ctx, review)
}

func (u *CatalogUsecase) GetProductReviews(ctx context.Context, productID string) ([]domain.Review, error) {
	return u.repo.GetReviews(ctx, productID)
}

// --- Collections ---

func (uc *CatalogUsecase) GetCollections(ctx context.Context) ([]domain.Collection, error) {
	return uc.repo.GetCollections(ctx)
}

func (uc *CatalogUsecase) GetAllCollections(ctx context.Context) ([]domain.Collection, error) {
	return uc.repo.GetAllCollections(ctx)
}

func (uc *CatalogUsecase) GetCollectionBySlug(ctx context.Context, slug string) (*domain.Collection, error) {
	return uc.repo.GetCollectionBySlug(ctx, slug)
}

func (uc *CatalogUsecase) CreateCollection(ctx context.Context, collection *domain.Collection) error {
	if collection.Title == "" {
		return fmt.Errorf("collection title is required")
	}
	if collection.Slug == "" {
		collection.Slug = utils.GenerateSlug(collection.Title)
	}
	if collection.ID == "" {
		collection.ID = utils.GenerateUUID()
	}
	// Removed manual override of IsActive. Frontend sends true by default. If false, it means Draft.
	// if !collection.IsActive {
	// 	collection.IsActive = true
	// }
	collection.CreatedAt = time.Now()
	collection.UpdatedAt = time.Now()
	return uc.repo.CreateCollection(ctx, collection)
}

func (uc *CatalogUsecase) UpdateCollection(ctx context.Context, collection *domain.Collection) error {
	collection.UpdatedAt = time.Now()
	return uc.repo.UpdateCollection(ctx, collection)
}

func (uc *CatalogUsecase) DeleteCollection(ctx context.Context, id string) error {
	return uc.repo.DeleteCollection(ctx, id)
}

func (uc *CatalogUsecase) GetProductStats(ctx context.Context) (*domain.ProductStats, error) {
	key := "admin:product_stats"
	if val, found := uc.cache.Get(key); found {
		return val.(*domain.ProductStats), nil
	}

	stats, err := uc.repo.GetProductStats(ctx)
	if err != nil {
		return nil, err
	}

	uc.cache.Set(key, stats, uc.cfg.CacheCategoryTTL) // Re-use category TTL or similar
	return stats, nil
}

func (uc *CatalogUsecase) AddProductToCollection(ctx context.Context, collectionID, productID string) error {
	return uc.repo.AddProductToCollection(ctx, collectionID, productID)
}

func (uc *CatalogUsecase) RemoveProductFromCollection(ctx context.Context, collectionID, productID string) error {
	return uc.repo.RemoveProductFromCollection(ctx, collectionID, productID)
}

func (uc *CatalogUsecase) invalidateStatsCache() {
	uc.cache.Delete("admin:product_stats")
}

func (uc *CatalogUsecase) invalidateCategoryCache() {
	uc.cache.Delete("category:tree:all")
	uc.cache.Delete("category:tree:nav")
	uc.cache.Delete("category:flat:all")
	uc.cache.Delete("category:flat:active")
	uc.cache.Delete("category:flat:inactive")
}

// GetVariantList returns a paginated list of variants with product context
func (u *CatalogUsecase) GetVariantList(ctx context.Context, filter domain.VariantListFilter) ([]domain.VariantWithProduct, domain.Pagination, error) {
	variants, count, err := u.repo.GetVariantList(ctx, filter)
	if err != nil {
		return nil, domain.Pagination{}, err
	}

	pagination := domain.Pagination{
		Page:       (filter.Offset / filter.Limit) + 1,
		Limit:      filter.Limit,
		TotalItems: count,
		TotalPages: int((count + int64(filter.Limit) - 1) / int64(filter.Limit)),
	}

	return variants, pagination, nil
}
