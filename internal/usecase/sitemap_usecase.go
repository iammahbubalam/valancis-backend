package usecase

import (
	"context"
	"fmt"
	"valancis-backend/config"
	"valancis-backend/internal/domain"
	"valancis-backend/pkg/cache"
	"time"
)

type SitemapItem struct {
	Loc        string
	LastMod    string
	ChangeFreq string
	Priority   float32
}

type SitemapUsecase struct {
	productRepo domain.ProductRepository
	baseURL     string
	cache       cache.CacheService
	cfg         *config.Config
}

func NewSitemapUsecase(repo domain.ProductRepository, baseURL string, cache cache.CacheService, cfg *config.Config) *SitemapUsecase {
	if baseURL == "" {
		// Should be handled by config, but strictly no hardcoded production domain here.
		// Fallback to empty string or a placeholder if really needed, but better to trust injection.
	}
	return &SitemapUsecase{
		productRepo: repo,
		baseURL:     baseURL,
		cache:       cache,
		cfg:         cfg,
	}
}

func (u *SitemapUsecase) GenerateSitemap(ctx context.Context) ([]SitemapItem, error) {
	key := "sitemap:items"
	if val, found := u.cache.Get(key); found {
		return val.([]SitemapItem), nil
	}

	var items []SitemapItem
	now := time.Now().Format("2006-01-02")

	// 1. Static Pages
	statics := []string{"", "/shop", "/collections", "/about", "/contact", "/login"} // Empty string for root
	for _, s := range statics {
		items = append(items, SitemapItem{
			Loc:        u.baseURL + s,
			LastMod:    now,
			ChangeFreq: "daily",
			Priority:   0.8,
		})
	}
	// Root has higher priority
	items[0].Priority = 1.0

	// 2. Products (Active only)
	isActive := true
	filter := domain.ProductFilter{
		Limit:    2000, // Reasonable limit for now
		Offset:   0,
		IsActive: &isActive,
	}
	products, _, err := u.productRepo.GetProducts(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}
	for _, p := range products {
		items = append(items, SitemapItem{
			Loc:        fmt.Sprintf("%s/product/%s", u.baseURL, p.Slug),
			LastMod:    p.UpdatedAt.Format("2006-01-02"),
			ChangeFreq: "weekly",
			Priority:   0.9,
		})
	}

	// 3. Categories
	categories, err := u.productRepo.GetCategoryTree(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}
	var flattenCats func([]domain.Category)
	flattenCats = func(cats []domain.Category) {
		for _, c := range cats {
			if c.IsActive {
				items = append(items, SitemapItem{
					Loc:        fmt.Sprintf("%s/category/%s", u.baseURL, c.Slug),
					LastMod:    now, // Categories don't track update time well in domain, use now
					ChangeFreq: "daily",
					Priority:   0.8,
				})
				if len(c.Children) > 0 {
					flattenCats(c.Children)
				}
			}
		}
	}
	flattenCats(categories)

	// 4. Collections
	collections, err := u.productRepo.GetCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch collections: %w", err)
	}
	for _, c := range collections {
		if c.IsActive {
			items = append(items, SitemapItem{
				Loc:        fmt.Sprintf("%s/collection/%s", u.baseURL, c.Slug),
				LastMod:    c.UpdatedAt.Format("2006-01-02"),
				ChangeFreq: "weekly",
				Priority:   0.8,
			})
		}
	}

	u.cache.Set(key, items, u.cfg.CacheSitemapTTL)
	return items, nil
}
