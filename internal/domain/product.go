package domain

import (
	"context"
	"time"
)

// --- Interfaces ---

type TransactionManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Category struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Slug            string     `json:"slug"`
	ParentID        *string    `json:"parentId"`
	Parent          *Category  `json:"-"`
	Children        []Category `json:"children"`
	OrderIndex      int        `json:"orderIndex"`
	Icon            string     `json:"icon"`
	Image           string     `json:"image"`
	IsFeatured      bool       `json:"isFeatured"`
	IsActive        bool       `json:"isActive"`
	ShowInNav       bool       `json:"showInNav"`
	MetaTitle       string     `json:"metaTitle"`
	MetaDescription string     `json:"metaDescription"`
	Keywords        string     `json:"keywords"`
	Products        []Product  `json:"products"`
}

type CategoryReorderItem struct {
	ID         string  `json:"id"`
	ParentID   *string `json:"parentId"`
	OrderIndex int     `json:"orderIndex"`
}

type Product struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Slug            string       `json:"slug"`
	Description     string       `json:"description"`
	BasePrice       float64      `json:"basePrice"`
	SalePrice       *float64     `json:"salePrice"`
	StockStatus     string       `json:"stockStatus"`
	Stock           int          `json:"stock"`
	IsFeatured      bool         `json:"isFeatured"`
	IsActive        bool         `json:"isActive"`
	Media           RawJSON      `json:"media"`
	Images          []string     `json:"images"` // Mapped from Media
	Attributes      JSONB        `json:"attributes"`
	Specs           JSONB        `json:"specifications"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
	Variants        []Variant    `json:"variants"`
	Categories      []Category   `json:"categories"`
	Collections     []Collection `json:"collections"`
	MetaTitle       string       `json:"metaTitle"`
	MetaDescription string       `json:"metaDescription"`
	Keywords        string       `json:"keywords"`
	OGImage         string       `json:"ogImage"`

	// L9 Fields
	Brand        string   `json:"brand"`
	Tags         []string `json:"tags"`
	WarrantyInfo JSONB    `json:"warrantyInfo"`
}

type Collection struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Slug            string    `json:"slug"`
	Description     string    `json:"description"`
	Image           string    `json:"image"`
	Story           string    `json:"story"` // The rich text narrative
	IsActive        bool      `json:"isActive"`
	MetaTitle       string    `json:"metaTitle"`
	MetaDescription string    `json:"metaDescription"`
	Keywords        string    `json:"keywords"`
	OGImage         string    `json:"ogImage"`
	Products        []Product `json:"products"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type Variant struct {
	ID        string `json:"id"`
	ProductID string `json:"productId"`
	Name      string `json:"name"`
	Stock     int    `json:"stock"`
	SKU       string `json:"sku"` // Optional: Variant specific SKU

	// L9 Fields
	Attributes        JSONB    `json:"attributes"`
	Price             *float64 `json:"price"` // Override base price
	SalePrice         *float64 `json:"salePrice"`
	Images            []string `json:"images"`
	Weight            *float64 `json:"weight"`
	Dimensions        JSONB    `json:"dimensions"`
	Barcode           string   `json:"barcode"`
	LowStockThreshold int      `json:"lowStockThreshold"`
}

// VariantWithProduct is used for SKU-level inventory listing
type VariantWithProduct struct {
	Variant
	ProductName      string  `json:"productName"`
	ProductSlug      string  `json:"productSlug"`
	ProductBasePrice float64 `json:"productBasePrice"`
	ProductImage     string  `json:"productImage"` // First image from product media
}

// VariantListFilter defines filters for variant listing
type VariantListFilter struct {
	ProductID    string
	LowStockOnly bool
	Search       string
	Sort         string // stock_asc, stock_desc, sku_asc
	Limit        int
	Offset       int
}

type ProductStats struct {
	TotalProducts       int64   `json:"totalProducts"`
	ActiveProducts      int64   `json:"activeProducts"`
	InactiveProducts    int64   `json:"inactiveProducts"`
	OutOfStock          int64   `json:"outOfStock"`
	LowStock            int64   `json:"lowStock"`
	TotalInventoryValue float64 `json:"totalInventoryValue"`
}

type InventoryLog struct {
	ID           uint      `json:"id"`
	ProductID    string    `json:"productId"`
	VariantID    *string   `json:"variantId"`
	ChangeAmount int       `json:"changeAmount"` // +10 or -5
	Reason       string    `json:"reason"`       // order_placed, restock, return, adjustment, cancelled
	ReferenceID  string    `json:"referenceId"`  // OrderID or Admin UserID
	CreatedAt    time.Time `json:"createdAt"`
}

// --- Interfaces ---

type ProductRepository interface {
	// Category Management
	GetCategoryTree(ctx context.Context) ([]Category, error)
	GetNavCategoryTree(ctx context.Context) ([]Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*Category, error)
	CreateCategory(ctx context.Context, category *Category) error
	UpdateCategory(ctx context.Context, category *Category) error
	DeleteCategory(ctx context.Context, id string) error
	ReorderCategories(ctx context.Context, updates []CategoryReorderItem) error
	GetCategoriesFlat(ctx context.Context, isActive *bool) ([]Category, error)

	// Collection Management
	GetCollections(ctx context.Context) ([]Collection, error)
	GetAllCollections(ctx context.Context) ([]Collection, error)
	GetCollectionBySlug(ctx context.Context, slug string) (*Collection, error)
	CreateCollection(ctx context.Context, collection *Collection) error
	UpdateCollection(ctx context.Context, collection *Collection) error
	DeleteCollection(ctx context.Context, id string) error
	AddProductToCollection(ctx context.Context, collectionID, productID string) error
	RemoveProductFromCollection(ctx context.Context, collectionID, productID string) error

	GetProducts(ctx context.Context, filter ProductFilter) ([]Product, int64, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	GetProductByID(ctx context.Context, id string) (*Product, error)
	UpdateStock(ctx context.Context, variantID string, quantity int, reason, referenceID string) error
	GetInventoryLogs(ctx context.Context, productID string, limit, offset int) ([]InventoryLog, int64, error)
	GetVariantList(ctx context.Context, filter VariantListFilter) ([]VariantWithProduct, int64, error)

	GetVariantByID(ctx context.Context, id string) (*Variant, error)
	// Admin Management
	CreateProduct(ctx context.Context, product *Product) error
	UpdateProduct(ctx context.Context, product *Product) error
	UpdateProductStatus(ctx context.Context, id string, isActive bool) error
	DeleteProduct(ctx context.Context, id string) error
	GetProductStats(ctx context.Context) (*ProductStats, error)

	// Reviews
	CreateReview(ctx context.Context, review *Review) error
	GetReviews(ctx context.Context, productID string) ([]Review, error)
}

type Review struct {
	ID        string    `json:"id"`
	ProductID string    `json:"productId"`
	UserID    string    `json:"userId"`
	User      User      `json:"user"`
	Rating    int       `json:"rating"` // 1-5
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"createdAt"`
}

type ProductFilter struct {
	CategorySlug string
	Query        string
	MinPrice     float64
	MaxPrice     float64
	Sort         string // newest, price_asc, price_desc
	Limit        int
	Offset       int
	IsActive     *bool // nil = all, true = active, false = inactive
	IsFeatured   *bool
}

// --- Custom Types moved to types.go ---
