package sqlcrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"valancis-backend/db/sqlc"
	"valancis-backend/internal/domain"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type productRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewProductRepository(db *pgxpool.Pool) domain.ProductRepository {
	return &productRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

// --- Helpers ---

func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	n.Scan(strconv.FormatFloat(f, 'f', -1, 64))
	return n
}

func float64PtrToNumeric(f *float64) pgtype.Numeric {
	var n pgtype.Numeric
	if f != nil {
		n.Scan(strconv.FormatFloat(*f, 'f', -1, 64))
	}
	return n
}

func numericToFloat64Ptr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f, _ := n.Float64Value()
	val := f.Float64
	return &val
}

// --- Mappers ---

func ptrStrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func sqlcProductToDomain(p sqlc.Product) domain.Product {
	prod := domain.Product{
		ID:              uuidToString(p.ID),
		Name:            p.Name,
		Slug:            p.Slug,
		Description:     ptrString(p.Description),
		BasePrice:       numericToFloat64(p.BasePrice),
		SalePrice:       numericToFloat64Ptr(p.SalePrice),
		StockStatus:     ptrString(p.StockStatus),
		IsFeatured:      p.IsFeatured,
		IsActive:        p.IsActive,
		CreatedAt:       pgtimeToTime(p.CreatedAt),
		UpdatedAt:       pgtimeToTime(p.UpdatedAt),
		MetaTitle:       ptrString(p.MetaTitle),
		MetaDescription: ptrString(p.MetaDescription),
		Keywords:        ptrString(p.MetaKeywords),
		OGImage:         ptrString(p.OgImage),
		Brand:           ptrString(p.Brand),
		Tags:            p.Tags,
	}

	// Handle Media (JSONB)
	if len(p.Media) > 0 {
		prod.Media = domain.RawJSON(p.Media)
		mapMediaToImages(&prod)
	}

	// Handle Attributes
	if len(p.Attributes) > 0 {
		var attrs domain.JSONB
		json.Unmarshal(p.Attributes, &attrs)
		prod.Attributes = attrs
	}

	// Handle Specs
	if len(p.Specifications) > 0 {
		var specs domain.JSONB
		json.Unmarshal(p.Specifications, &specs)
		prod.Specs = specs
	}

	// Handle Warranty Info
	if len(p.WarrantyInfo) > 0 {
		var warranty domain.JSONB
		json.Unmarshal(p.WarrantyInfo, &warranty)
		prod.WarrantyInfo = warranty
	}

	return prod
}

func mapMediaToImages(p *domain.Product) {
	if len(p.Media) == 0 {
		return
	}
	var arr []string
	if err := json.Unmarshal([]byte(p.Media), &arr); err == nil {
		p.Images = arr
		return
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(p.Media), &obj); err == nil {
		if imgs, ok := obj["images"].([]interface{}); ok {
			p.Images = make([]string, len(imgs))
			for i, v := range imgs {
				if str, ok := v.(string); ok {
					p.Images[i] = str
				}
			}
		}
	}
}

func mapImagesToMedia(p *domain.Product) []byte {
	if p.Images != nil {
		bytes, _ := json.Marshal(p.Images)
		return bytes
	}
	return nil
}

func sqlcCategoryToDomain(c sqlc.Category) domain.Category {
	var parentID *string
	if c.ParentID.Valid {
		pid := uuidToString(c.ParentID)
		parentID = &pid
	}
	return domain.Category{
		ID:              uuidToString(c.ID),
		Name:            ptrStrToStr(c.Name),
		Slug:            ptrStrToStr(c.Slug),
		ParentID:        parentID,
		OrderIndex:      int(c.OrderIndex),
		Icon:            ptrString(c.Icon),
		Image:           ptrString(c.Image),
		IsActive:        c.IsActive,
		ShowInNav:       c.ShowInNav,
		IsFeatured:      c.IsFeatured,
		MetaTitle:       ptrString(c.MetaTitle),
		MetaDescription: ptrString(c.MetaDescription),
		Keywords:        ptrString(c.Keywords),
	}
}

func sqlcCollectionToDomain(c sqlc.Collection) domain.Collection {
	return domain.Collection{
		ID:              uuidToString(c.ID),
		Title:           c.Title,
		Slug:            c.Slug,
		Description:     ptrString(c.Description),
		Image:           ptrString(c.Image),
		Story:           ptrString(c.Story),
		IsActive:        c.IsActive,
		CreatedAt:       pgtimeToTime(c.CreatedAt),
		UpdatedAt:       pgtimeToTime(c.UpdatedAt),
		MetaTitle:       ptrString(c.MetaTitle),
		MetaDescription: ptrString(c.MetaDescription),
		Keywords:        ptrString(c.MetaKeywords),
		OGImage:         ptrString(c.OgImage),
	}
}

func sqlcVariantToDomain(v sqlc.Variant) domain.Variant {
	variant := domain.Variant{
		ID:                uuidToString(v.ID),
		ProductID:         uuidToString(v.ProductID),
		Name:              v.Name,
		Stock:             int(v.Stock),
		SKU:               ptrString(v.Sku),
		Price:             numericToFloat64Ptr(v.Price),
		SalePrice:         numericToFloat64Ptr(v.SalePrice),
		Images:            v.Images,
		Weight:            numericToFloat64Ptr(v.Weight),
		Barcode:           ptrString(v.Barcode),
		LowStockThreshold: int(v.LowStockThreshold),
	}

	if len(v.Attributes) > 0 {
		var attrs domain.JSONB
		json.Unmarshal(v.Attributes, &attrs)
		variant.Attributes = attrs
	}

	if len(v.Dimensions) > 0 {
		var dims domain.JSONB
		json.Unmarshal(v.Dimensions, &dims)
		variant.Dimensions = dims
	}

	return variant
}

func sqlcInventoryLogToDomain(l sqlc.InventoryLog) domain.InventoryLog {
	var variantID *string
	if l.VariantID.Valid {
		vid := uuidToString(l.VariantID)
		variantID = &vid
	}
	return domain.InventoryLog{
		ID:           uint(l.ID),
		ProductID:    uuidToString(l.ProductID),
		VariantID:    variantID,
		ChangeAmount: int(l.ChangeAmount),
		Reason:       l.Reason,
		ReferenceID:  l.ReferenceID,
		CreatedAt:    pgtimeToTime(l.CreatedAt),
	}
}

func sqlcReviewToDomain(r sqlc.Review) domain.Review {
	return domain.Review{
		ID:        uuidToString(r.ID),
		ProductID: uuidToString(r.ProductID),
		UserID:    uuidToString(r.UserID),
		Rating:    int(r.Rating),
		Comment:   ptrString(r.Comment),
		CreatedAt: pgtimeToTime(r.CreatedAt),
	}
}

// --- Category Methods ---

func (r *productRepository) GetCategoryTree(ctx context.Context) ([]domain.Category, error) {
	return r.buildCategoryTree(ctx, false)
}

func (r *productRepository) GetNavCategoryTree(ctx context.Context) ([]domain.Category, error) {
	return r.buildCategoryTree(ctx, true)
}

func (r *productRepository) buildCategoryTree(ctx context.Context, navOnly bool) ([]domain.Category, error) {
	var roots []sqlc.Category
	var err error

	if navOnly {
		roots, err = r.queries.GetActiveNavCategories(ctx)
	} else {
		roots, err = r.queries.GetRootCategories(ctx)
	}
	if err != nil {
		return nil, err
	}

	result := make([]domain.Category, len(roots))
	for i, root := range roots {
		cat := sqlcCategoryToDomain(root)
		cat.Children, _ = r.getChildrenRecursive(ctx, root.ID, navOnly, 3)
		result[i] = cat
	}
	return result, nil
}

func (r *productRepository) getChildrenRecursive(ctx context.Context, parentID pgtype.UUID, navOnly bool, depth int) ([]domain.Category, error) {
	if depth <= 0 {
		return nil, nil
	}

	var children []sqlc.Category
	var err error

	if navOnly {
		children, err = r.queries.GetActiveChildCategories(ctx, parentID)
	} else {
		children, err = r.queries.GetChildCategories(ctx, parentID)
	}
	if err != nil {
		return nil, err
	}

	result := make([]domain.Category, len(children))
	for i, child := range children {
		cat := sqlcCategoryToDomain(child)
		cat.Children, _ = r.getChildrenRecursive(ctx, child.ID, navOnly, depth-1)
		result[i] = cat
	}
	return result, nil
}

func (r *productRepository) GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	c, err := r.queries.GetCategoryBySlug(ctx, &slug)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	cat := sqlcCategoryToDomain(c)
	return &cat, nil
}

// GetCategoriesFlat returns a flat list of categories (no hierarchy) with optional isActive filter
func (r *productRepository) GetCategoriesFlat(ctx context.Context, isActive *bool) ([]domain.Category, error) {
	cats, err := r.queries.GetCategoriesFlat(ctx, isActive)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Category, 0, len(cats))
	for _, c := range cats {
		// Skip empty/invalid categories
		if c.Name == nil || *c.Name == "" {
			continue
		}
		result = append(result, sqlcCategoryToDomain(c))
	}
	return result, nil
}

func (r *productRepository) CreateCategory(ctx context.Context, category *domain.Category) error {
	var parentID pgtype.UUID
	if category.ParentID != nil {
		parentID = stringToUUID(*category.ParentID)
	}

	created, err := r.queries.CreateCategory(ctx, sqlc.CreateCategoryParams{
		Name:            strPtr(category.Name),
		Slug:            strPtr(category.Slug),
		ParentID:        parentID,
		OrderIndex:      int32(category.OrderIndex),
		Icon:            strPtr(category.Icon),
		Image:           strPtr(category.Image),
		IsActive:        category.IsActive,
		ShowInNav:       category.ShowInNav,
		MetaTitle:       strPtr(category.MetaTitle),
		MetaDescription: strPtr(category.MetaDescription),
		Keywords:        strPtr(category.Keywords),
		IsFeatured:      category.IsFeatured,
	})
	if err != nil {
		return err
	}
	category.ID = uuidToString(created.ID)
	return nil
}

func (r *productRepository) UpdateCategory(ctx context.Context, category *domain.Category) error {
	var parentID pgtype.UUID
	if category.ParentID != nil {
		parentID = stringToUUID(*category.ParentID)
	}

	_, err := r.queries.UpdateCategory(ctx, sqlc.UpdateCategoryParams{
		ID:              stringToUUID(category.ID),
		Name:            strPtr(category.Name),
		Slug:            strPtr(category.Slug),
		ParentID:        parentID,
		OrderIndex:      int32(category.OrderIndex),
		Icon:            strPtr(category.Icon),
		Image:           strPtr(category.Image),
		IsActive:        category.IsActive,
		ShowInNav:       category.ShowInNav,
		MetaTitle:       strPtr(category.MetaTitle),
		MetaDescription: strPtr(category.MetaDescription),
		Keywords:        strPtr(category.Keywords),
		IsFeatured:      category.IsFeatured,
	})
	return err
}

func (r *productRepository) DeleteCategory(ctx context.Context, id string) error {
	return r.queries.DeleteCategory(ctx, stringToUUID(id))
}

func (r *productRepository) ReorderCategories(ctx context.Context, updates []domain.CategoryReorderItem) error {
	for _, item := range updates {
		var parentID pgtype.UUID
		if item.ParentID != nil {
			parentID = stringToUUID(*item.ParentID)
		}
		err := r.queries.UpdateCategoryOrder(ctx, sqlc.UpdateCategoryOrderParams{
			ID:         stringToUUID(item.ID),
			OrderIndex: int32(item.OrderIndex),
			ParentID:   parentID,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// --- Collection Methods ---

func (r *productRepository) GetCollections(ctx context.Context) ([]domain.Collection, error) {
	cols, err := r.queries.GetActiveCollections(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Collection, len(cols))
	for i, c := range cols {
		result[i] = sqlcCollectionToDomain(c)
	}
	return result, nil
}

func (r *productRepository) GetAllCollections(ctx context.Context) ([]domain.Collection, error) {
	cols, err := r.queries.GetAllCollections(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Collection, len(cols))
	for i, c := range cols {
		result[i] = sqlcCollectionToDomain(c)
	}
	return result, nil
}

func (r *productRepository) GetCollectionBySlug(ctx context.Context, slug string) (*domain.Collection, error) {
	col, err := r.queries.GetCollectionBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	collection := sqlcCollectionToDomain(col)

	// Get products for collection
	products, err := r.queries.GetProductsForCollection(ctx, col.ID)
	if err == nil && len(products) > 0 {
		collection.Products = make([]domain.Product, len(products))
		productIDs := make([]pgtype.UUID, len(products))
		for i, p := range products {
			productIDs[i] = p.ID
			collection.Products[i] = sqlcProductToDomain(p)
		}
		// Enrich with variants to calculate correct stock
		r.enrichVariants(ctx, collection.Products, productIDs)
	}

	return &collection, nil
}

func (r *productRepository) CreateCollection(ctx context.Context, collection *domain.Collection) error {
	created, err := r.queries.CreateCollection(ctx, sqlc.CreateCollectionParams{
		Title:           collection.Title,
		Slug:            collection.Slug,
		Description:     strPtr(collection.Description),
		Image:           strPtr(collection.Image),
		Story:           strPtr(collection.Story),
		IsActive:        collection.IsActive,
		MetaTitle:       strPtr(collection.MetaTitle),
		MetaDescription: strPtr(collection.MetaDescription),
		MetaKeywords:    strPtr(collection.Keywords),
		OgImage:         strPtr(collection.OGImage),
	})
	if err != nil {
		return err
	}
	collection.ID = uuidToString(created.ID)
	collection.CreatedAt = pgtimeToTime(created.CreatedAt)
	collection.UpdatedAt = pgtimeToTime(created.UpdatedAt)
	return nil
}

func (r *productRepository) UpdateCollection(ctx context.Context, collection *domain.Collection) error {
	_, err := r.queries.UpdateCollection(ctx, sqlc.UpdateCollectionParams{
		ID:              stringToUUID(collection.ID),
		Title:           collection.Title,
		Slug:            collection.Slug,
		Description:     strPtr(collection.Description),
		Image:           strPtr(collection.Image),
		Story:           strPtr(collection.Story),
		IsActive:        collection.IsActive,
		MetaTitle:       strPtr(collection.MetaTitle),
		MetaDescription: strPtr(collection.MetaDescription),
		MetaKeywords:    strPtr(collection.Keywords),
		OgImage:         strPtr(collection.OGImage),
	})
	return err
}

func (r *productRepository) DeleteCollection(ctx context.Context, id string) error {
	return r.queries.DeleteCollection(ctx, stringToUUID(id))
}

func (r *productRepository) AddProductToCollection(ctx context.Context, collectionID, productID string) error {
	return r.queries.AddProductToCollection(ctx, sqlc.AddProductToCollectionParams{
		ProductID:    stringToUUID(productID),
		CollectionID: stringToUUID(collectionID),
	})
}

func (r *productRepository) RemoveProductFromCollection(ctx context.Context, collectionID, productID string) error {
	return r.queries.RemoveProductFromCollection(ctx, sqlc.RemoveProductFromCollectionParams{
		ProductID:    stringToUUID(productID),
		CollectionID: stringToUUID(collectionID),
	})
}

// --- Product Methods ---

func (r *productRepository) GetProducts(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, int64, error) {
	limit := int32(filter.Limit)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Default to true if nil (show active products by default) IS REMOVED
	// We want to allow nil to mean "ALL"
	// The caller (Public API) should ensure IsActive is set to True if needed.
	// Admin API will pass nil for All, True for Active, False for Inactive.

	var products []sqlc.Product
	var count int64
	var err error

	// Priority 1: Full-text search query (if provided)
	if filter.Query != "" {
		// Use full-text search queries
		searchRows, err := r.queries.SearchProducts(ctx, sqlc.SearchProductsParams{
			Query:    filter.Query,
			IsActive: filter.IsActive,
			Limit:    limit,
			Offset:   int32(filter.Offset),
		})
		if err != nil {
			return nil, 0, err
		}

		count, err = r.queries.CountSearchProducts(ctx, sqlc.CountSearchProductsParams{
			Query:    filter.Query,
			IsActive: filter.IsActive,
		})
		if err != nil {
			return nil, 0, err
		}

		// Convert search rows to standard product rows for unified processing
		for _, row := range searchRows {
			products = append(products, sqlc.Product{
				ID:   row.ID,
				Name: row.Name,
				Slug: row.Slug,
				// SKU, Stock, LowStockThreshold moved to variants
				Description: row.Description,
				BasePrice:   row.BasePrice,
				SalePrice:   row.SalePrice,
				// SKU, Stock, LowStockThreshold moved to variants
				IsFeatured:      row.IsFeatured,
				IsActive:        row.IsActive,
				Media:           row.Media,
				Attributes:      row.Attributes,
				Specifications:  row.Specifications,
				MetaTitle:       row.MetaTitle,
				MetaDescription: row.MetaDescription,
				MetaKeywords:    row.MetaKeywords,
				OgImage:         row.OgImage,
				Brand:           row.Brand,
				Tags:            row.Tags,
				WarrantyInfo:    row.WarrantyInfo,
				CreatedAt:       row.CreatedAt,
				UpdatedAt:       row.UpdatedAt,
			})
		}
	} else if filter.CategorySlug != "" {
		// Priority 2: Category filter
		products, err = r.queries.GetProductsWithCategoryFilter(ctx, sqlc.GetProductsWithCategoryFilterParams{
			Slug:     strPtr(filter.CategorySlug),
			IsActive: filter.IsActive,
			Limit:    limit,
			Offset:   int32(filter.Offset),
		})
		if err != nil {
			return nil, 0, err
		}

		count, err = r.queries.CountProductsWithCategoryFilter(ctx, sqlc.CountProductsWithCategoryFilterParams{
			Slug:     strPtr(filter.CategorySlug),
			IsActive: filter.IsActive,
		})
		if err != nil {
			return nil, 0, err
		}
	} else {
		// Default: Standard product listing
		products, err = r.queries.GetProducts(ctx, sqlc.GetProductsParams{
			IsActive:   filter.IsActive,
			IsFeatured: filter.IsFeatured,
			Limit:      limit,
			Offset:     int32(filter.Offset),
		})
		if err != nil {
			return nil, 0, err
		}

		count, err = r.queries.CountProducts(ctx, sqlc.CountProductsParams{
			IsActive:   filter.IsActive,
			IsFeatured: filter.IsFeatured,
		})
		if err != nil {
			return nil, 0, err
		}
	}

	result := make([]domain.Product, len(products))
	if len(products) > 0 {
		productIDs := make([]pgtype.UUID, len(products))
		for i, p := range products {
			productIDs[i] = p.ID
			result[i] = sqlcProductToDomain(p)
		}

		// L9 OPTIMIZATION: Hydrate relations in parallel to reduce total latency
		var wg sync.WaitGroup
		wg.Add(3)

		go func() {
			defer wg.Done()
			r.enrichVariants(ctx, result, productIDs)
		}()

		go func() {
			defer wg.Done()
			r.enrichCategories(ctx, result, productIDs)
		}()

		go func() {
			defer wg.Done()
			r.enrichCollections(ctx, result, productIDs)
		}()
		wg.Wait()
	}
	return result, count, nil
}

// --- Hydrators (L9 Standard: Composable & Reusable) ---

func (r *productRepository) enrichVariants(ctx context.Context, products []domain.Product, productIDs []pgtype.UUID) {
	variantRows, err := r.queries.GetVariantsByProductIDs(ctx, productIDs)
	if err == nil && len(variantRows) > 0 {
		prodVarMap := make(map[string][]domain.Variant)
		prodStockMap := make(map[string]int)

		for _, v := range variantRows {
			if v.ProductID.Valid {
				pid := uuidToString(v.ProductID)
				variant := sqlcVariantToDomain(v)
				prodVarMap[pid] = append(prodVarMap[pid], variant)
				prodStockMap[pid] += variant.Stock
			}
		}

		for i := range products {
			pid := products[i].ID
			if vars, ok := prodVarMap[pid]; ok {
				products[i].Variants = vars
				products[i].Stock = prodStockMap[pid]

				// L9: Sync StockStatus if it's nil or out of sync
				if products[i].StockStatus == "" || products[i].StockStatus == "in_stock" {
					if products[i].Stock <= 0 {
						products[i].StockStatus = "out_of_stock"
					} else {
						products[i].StockStatus = "in_stock"
					}
				}
			} else {
				products[i].Variants = []domain.Variant{}
				products[i].Stock = 0
				products[i].StockStatus = "out_of_stock"
			}
		}
	} else {
		for i := range products {
			products[i].Variants = []domain.Variant{}
			products[i].Stock = 0
			products[i].StockStatus = "out_of_stock"
		}
	}
}

func (r *productRepository) enrichCategories(ctx context.Context, products []domain.Product, productIDs []pgtype.UUID) {
	catRows, err := r.queries.GetCategoryIDsForProducts(ctx, productIDs)
	if err == nil && len(catRows) > 0 {
		catIDSet := make(map[pgtype.UUID]struct{})
		for _, row := range catRows {
			if row.CategoryID.Valid {
				catIDSet[row.CategoryID] = struct{}{}
			}
		}
		uniqueCatIDs := make([]pgtype.UUID, 0, len(catIDSet))
		for id := range catIDSet {
			uniqueCatIDs = append(uniqueCatIDs, id)
		}

		cats, err := r.queries.GetCategoriesByIDs(ctx, uniqueCatIDs)
		if err == nil {
			catMap := make(map[string]domain.Category)
			for _, c := range cats {
				catMap[uuidToString(c.ID)] = sqlcCategoryToDomain(c)
			}
			prodCatMap := make(map[string][]domain.Category)
			for _, row := range catRows {
				if row.CategoryID.Valid {
					pid := uuidToString(row.ProductID)
					cid := uuidToString(row.CategoryID)
					if c, ok := catMap[cid]; ok {
						prodCatMap[pid] = append(prodCatMap[pid], c)
					}
				}
			}
			for i := range products {
				if cs, ok := prodCatMap[products[i].ID]; ok {
					products[i].Categories = cs
				} else {
					products[i].Categories = []domain.Category{}
				}
			}
			return
		}
	}
	// Fallback/Empty
	for i := range products {
		products[i].Categories = []domain.Category{}
	}
}

func (r *productRepository) enrichCollections(ctx context.Context, products []domain.Product, productIDs []pgtype.UUID) {
	colRows, err := r.queries.GetCollectionIDsForProducts(ctx, productIDs)
	if err == nil && len(colRows) > 0 {
		colIDSet := make(map[pgtype.UUID]struct{})
		for _, row := range colRows {
			if row.CollectionID.Valid {
				colIDSet[row.CollectionID] = struct{}{}
			}
		}
		uniqueColIDs := make([]pgtype.UUID, 0, len(colIDSet))
		for id := range colIDSet {
			uniqueColIDs = append(uniqueColIDs, id)
		}
		cols, err := r.queries.GetCollectionsByIDs(ctx, uniqueColIDs)
		if err == nil {
			colMap := make(map[string]domain.Collection)
			for _, c := range cols {
				colMap[uuidToString(c.ID)] = sqlcCollectionToDomain(c)
			}
			prodColMap := make(map[string][]domain.Collection)
			for _, row := range colRows {
				if row.CollectionID.Valid {
					pid := uuidToString(row.ProductID)
					cid := uuidToString(row.CollectionID)
					if c, ok := colMap[cid]; ok {
						prodColMap[pid] = append(prodColMap[pid], c)
					}
				}
			}
			for i := range products {
				if cs, ok := prodColMap[products[i].ID]; ok {
					products[i].Collections = cs
				} else {
					products[i].Collections = []domain.Collection{}
				}
			}
			return
		}
	}
	// Fallback/Empty
	for i := range products {
		products[i].Collections = []domain.Collection{}
	}
}

func (r *productRepository) GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	p, err := r.queries.GetProductBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	prod := sqlcProductToDomain(p)

	// Load variants
	variants, _ := r.queries.GetVariantsByProductID(ctx, p.ID)
	prod.Variants = make([]domain.Variant, len(variants))
	totalStock := 0
	for i, v := range variants {
		variant := sqlcVariantToDomain(v)
		prod.Variants[i] = variant
		totalStock += variant.Stock
	}
	prod.Stock = totalStock

	// Sync StockStatus
	if prod.StockStatus == "" || prod.StockStatus == "in_stock" {
		if prod.Stock <= 0 {
			prod.StockStatus = "out_of_stock"
		} else {
			prod.StockStatus = "in_stock"
		}
	}

	// Load categories (Optimized Batch Fetch)
	catIDs, _ := r.queries.GetCategoryIDsForProduct(ctx, p.ID)
	if len(catIDs) > 0 {
		cats, err := r.queries.GetCategoriesByIDs(ctx, catIDs)
		if err == nil {
			prod.Categories = make([]domain.Category, len(cats))
			for i, c := range cats {
				prod.Categories[i] = sqlcCategoryToDomain(c)
			}
		} else {
			prod.Categories = []domain.Category{}
		}
	} else {
		prod.Categories = []domain.Category{}
	}

	// Load Collections (Optimized Batch Fetch)
	colIDs, _ := r.queries.GetCollectionIDsForProduct(ctx, p.ID)
	if len(colIDs) > 0 {
		cols, err := r.queries.GetCollectionsByIDs(ctx, colIDs)
		if err == nil {
			prod.Collections = make([]domain.Collection, len(cols))
			for i, c := range cols {
				prod.Collections[i] = sqlcCollectionToDomain(c)
			}
		} else {
			prod.Collections = []domain.Collection{}
		}
	} else {
		prod.Collections = []domain.Collection{}
	}

	return &prod, nil
}

func (r *productRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	p, err := r.queries.GetProductByID(ctx, stringToUUID(id))
	if err != nil {
		return nil, err
	}
	prod := sqlcProductToDomain(p)

	// Load variants
	variants, _ := r.queries.GetVariantsByProductID(ctx, p.ID)
	prod.Variants = make([]domain.Variant, len(variants))
	totalStock := 0
	for i, v := range variants {
		variant := sqlcVariantToDomain(v)
		prod.Variants[i] = variant
		totalStock += variant.Stock
	}
	prod.Stock = totalStock

	// Sync StockStatus
	if prod.StockStatus == "" || prod.StockStatus == "in_stock" {
		if prod.Stock <= 0 {
			prod.StockStatus = "out_of_stock"
		} else {
			prod.StockStatus = "in_stock"
		}
	}

	// Load categories
	catIDs, _ := r.queries.GetCategoryIDsForProduct(ctx, p.ID)
	if len(catIDs) > 0 {
		cats, err := r.queries.GetCategoriesByIDs(ctx, catIDs)
		if err == nil {
			prod.Categories = make([]domain.Category, len(cats))
			for i, c := range cats {
				prod.Categories[i] = sqlcCategoryToDomain(c)
			}
		} else {
			prod.Categories = []domain.Category{}
		}
	} else {
		prod.Categories = []domain.Category{}
	}

	// Load Collections
	colIDs, _ := r.queries.GetCollectionIDsForProduct(ctx, p.ID)
	if len(colIDs) > 0 {
		cols, err := r.queries.GetCollectionsByIDs(ctx, colIDs)
		if err == nil {
			prod.Collections = make([]domain.Collection, len(cols))
			for i, c := range cols {
				prod.Collections[i] = sqlcCollectionToDomain(c)
			}
		} else {
			prod.Collections = []domain.Collection{}
		}
	} else {
		prod.Collections = []domain.Collection{}
	}

	return &prod, nil
}

func (r *productRepository) UpdateStock(ctx context.Context, variantID string, quantity int, reason, referenceID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// 1. Get Variant to confirm existence and get ProductID
	targetUUID := stringToUUID(variantID)
	v, err := qtx.GetVariantByID(ctx, targetUUID)
	if err != nil {
		return fmt.Errorf("variant not found: %s", variantID)
	}

	// 2. Update variant stock
	rows, err := qtx.UpdateVariantStock(ctx, sqlc.UpdateVariantStockParams{
		ID:    targetUUID,
		Stock: int32(quantity),
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("insufficient stock for variant: %s", variantID)
	}

	// 3. Create log
	_, err = qtx.CreateInventoryLog(ctx, sqlc.CreateInventoryLogParams{
		ProductID:    v.ProductID,
		VariantID:    targetUUID,
		ChangeAmount: int32(quantity),
		Reason:       reason,
		ReferenceID:  referenceID,
	})
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *productRepository) GetInventoryLogs(ctx context.Context, productID string, limit, offset int) ([]domain.InventoryLog, int64, error) {
	var prodUUID pgtype.UUID
	if productID != "" {
		prodUUID = stringToUUID(productID)
	}

	logs, err := r.queries.GetInventoryLogs(ctx, sqlc.GetInventoryLogsParams{
		Column1: prodUUID,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := r.queries.CountInventoryLogs(ctx, prodUUID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]domain.InventoryLog, len(logs))
	for i, l := range logs {
		result[i] = sqlcInventoryLogToDomain(l)
	}

	return result, count, nil
}

// --- Admin Product Methods ---

func (r *productRepository) CreateProduct(ctx context.Context, product *domain.Product) error {
	if product.CreatedAt.IsZero() {
		product.CreatedAt = time.Now()
	}
	if product.UpdatedAt.IsZero() {
		product.UpdatedAt = time.Now()
	}

	mediaBytes := mapImagesToMedia(product)
	attrsBytes, _ := json.Marshal(product.Attributes)
	specsBytes, _ := json.Marshal(product.Specs)

	// Marshal additional JSONB fields
	warrantyBytes, _ := json.Marshal(product.WarrantyInfo)

	// Start transaction for atomic Product + Master Variant creation
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	created, err := qtx.CreateProduct(ctx, sqlc.CreateProductParams{
		Name:            product.Name,
		Slug:            product.Slug,
		Description:     strPtr(product.Description),
		BasePrice:       float64ToNumeric(product.BasePrice),
		SalePrice:       float64PtrToNumeric(product.SalePrice),
		StockStatus:     strPtr(product.StockStatus),
		IsFeatured:      product.IsFeatured,
		IsActive:        product.IsActive,
		Media:           mediaBytes,
		Attributes:      attrsBytes,
		Specifications:  specsBytes,
		MetaTitle:       strPtr(product.MetaTitle),
		MetaDescription: strPtr(product.MetaDescription),
		MetaKeywords:    strPtr(product.Keywords),
		OgImage:         strPtr(product.OGImage),
		Brand:           strPtr(product.Brand),
		Tags:            product.Tags,
		WarrantyInfo:    warrantyBytes,
	})
	if err != nil {
		return err
	}

	product.ID = uuidToString(created.ID)
	product.CreatedAt = pgtimeToTime(created.CreatedAt)
	product.UpdatedAt = pgtimeToTime(created.UpdatedAt)

	// Add Master Variant if none exist
	// This ensures Phase 6 SSOT is maintained
	if len(product.Variants) == 0 {
		_, err = qtx.CreateVariant(ctx, sqlc.CreateVariantParams{
			ProductID:         created.ID,
			Name:              "Default",
			Stock:             0, // Initial stock handled by variant creation later if needed
			LowStockThreshold: 5,
			Attributes:        []byte("{}"),
		})
		if err != nil {
			return err
		}
	} else {
		for _, v := range product.Variants {
			vAttrs, _ := json.Marshal(v.Attributes)
			_, err = qtx.CreateVariant(ctx, sqlc.CreateVariantParams{
				ProductID:         created.ID,
				Name:              v.Name,
				Stock:             int32(v.Stock),
				Sku:               strPtr(v.SKU),
				Price:             float64PtrToNumeric(v.Price),
				SalePrice:         float64PtrToNumeric(v.SalePrice),
				Images:            v.Images,
				Weight:            float64PtrToNumeric(v.Weight),
				Barcode:           strPtr(v.Barcode),
				Attributes:        vAttrs,
				LowStockThreshold: int32(v.LowStockThreshold),
			})
			if err != nil {
				return err
			}
		}
	}

	// Add categories
	for _, cat := range product.Categories {
		qtx.AddProductCategory(ctx, sqlc.AddProductCategoryParams{
			ProductID:  created.ID,
			CategoryID: stringToUUID(cat.ID),
		})
	}

	// Add collections
	for _, col := range product.Collections {
		qtx.AddProductCollection(ctx, sqlc.AddProductCollectionParams{
			ProductID:    created.ID,
			CollectionID: stringToUUID(col.ID),
		})
	}

	return tx.Commit(ctx)
}

func (r *productRepository) UpdateProduct(ctx context.Context, product *domain.Product) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	productUUID := stringToUUID(product.ID)
	mediaBytes := mapImagesToMedia(product)
	attrsBytes, _ := json.Marshal(product.Attributes)
	specsBytes, _ := json.Marshal(product.Specs)
	warrantyBytes, _ := json.Marshal(product.WarrantyInfo)

	_, err = qtx.UpdateProduct(ctx, sqlc.UpdateProductParams{
		ID:              productUUID,
		Name:            product.Name,
		Slug:            product.Slug,
		Description:     strPtr(product.Description),
		BasePrice:       float64ToNumeric(product.BasePrice),
		SalePrice:       float64PtrToNumeric(product.SalePrice),
		StockStatus:     strPtr(product.StockStatus),
		IsFeatured:      product.IsFeatured,
		IsActive:        product.IsActive,
		Media:           mediaBytes,
		Attributes:      attrsBytes,
		Specifications:  specsBytes,
		MetaTitle:       strPtr(product.MetaTitle),
		MetaDescription: strPtr(product.MetaDescription),
		MetaKeywords:    strPtr(product.Keywords),
		OgImage:         strPtr(product.OGImage),
		Brand:           strPtr(product.Brand),
		Tags:            product.Tags,
		WarrantyInfo:    warrantyBytes,
	})
	if err != nil {
		return err
	}

	// Update categories
	qtx.ClearProductCategories(ctx, productUUID)
	for _, cat := range product.Categories {
		qtx.AddProductCategory(ctx, sqlc.AddProductCategoryParams{
			ProductID:  productUUID,
			CategoryID: stringToUUID(cat.ID),
		})
	}

	// Update collections
	qtx.ClearProductCollections(ctx, productUUID)
	for _, col := range product.Collections {
		qtx.AddProductCollection(ctx, sqlc.AddProductCollectionParams{
			ProductID:    productUUID,
			CollectionID: stringToUUID(col.ID),
		})
	}

	// Update variants (Smart Update Strategy: Sync)
	// 1. Fetch existing variants
	existingVariants, err := qtx.GetVariantsByProductID(ctx, productUUID)
	if err != nil {
		return err
	}
	existingVarMap := make(map[string]struct{})
	for _, v := range existingVariants {
		existingVarMap[uuidToString(v.ID)] = struct{}{}
	}
	// Track processed IDs to identify deletions
	processedIDs := make(map[string]struct{})

	for _, v := range product.Variants {
		vAttributes, _ := json.Marshal(v.Attributes)
		vDimensions, _ := json.Marshal(v.Dimensions)

		if v.ID != "" {
			// Check if it exists in DB
			if _, exists := existingVarMap[v.ID]; exists {
				// UPDATE
				_, err := qtx.UpdateVariant(ctx, sqlc.UpdateVariantParams{
					ID:                stringToUUID(v.ID),
					Name:              v.Name,
					Stock:             int32(v.Stock),
					Sku:               strPtr(v.SKU),
					Attributes:        vAttributes,
					Price:             float64PtrToNumeric(v.Price),
					SalePrice:         float64PtrToNumeric(v.SalePrice),
					Images:            v.Images,
					Weight:            float64PtrToNumeric(v.Weight),
					Dimensions:        vDimensions,
					Barcode:           strPtr(v.Barcode),
					LowStockThreshold: int32(v.LowStockThreshold),
				})
				if err != nil {
					return fmt.Errorf("failed to update variant %s: %w", v.ID, err)
				}
				processedIDs[v.ID] = struct{}{}
				continue
			}
		}

		// CREATE (if ID is empty or not found in DB)
		_, err = qtx.CreateVariant(ctx, sqlc.CreateVariantParams{
			ProductID:         productUUID,
			Name:              v.Name,
			Stock:             int32(v.Stock),
			Sku:               strPtr(v.SKU),
			Attributes:        vAttributes,
			Price:             float64PtrToNumeric(v.Price),
			SalePrice:         float64PtrToNumeric(v.SalePrice),
			Images:            v.Images,
			Weight:            float64PtrToNumeric(v.Weight),
			Dimensions:        vDimensions,
			Barcode:           strPtr(v.Barcode),
			LowStockThreshold: int32(v.LowStockThreshold),
		})
		if err != nil {
			return fmt.Errorf("failed to create variant %s: %w", v.Name, err)
		}
	}

	// DELETE orphans (variants in DB but not in payload)
	for id := range existingVarMap {
		if _, processed := processedIDs[id]; !processed {
			if err := qtx.DeleteVariant(ctx, stringToUUID(id)); err != nil {
				return fmt.Errorf("failed to delete orphan variant %s: %w", id, err)
			}
		}
	}

	return tx.Commit(ctx)
}

func (r *productRepository) UpdateProductStatus(ctx context.Context, id string, isActive bool) error {
	return r.queries.UpdateProductStatus(ctx, sqlc.UpdateProductStatusParams{
		ID:       stringToUUID(id),
		IsActive: isActive,
	})
}

func (r *productRepository) DeleteProduct(ctx context.Context, id string) error {
	return r.queries.DeleteProduct(ctx, stringToUUID(id))
}

func (r *productRepository) GetVariantList(ctx context.Context, filter domain.VariantListFilter) ([]domain.VariantWithProduct, int64, error) {
	var prodUUID pgtype.UUID
	if filter.ProductID != "" {
		prodUUID = stringToUUID(filter.ProductID)
	}

	arg := sqlc.GetAllVariantsWithProductParams{
		Column1: prodUUID,
		Column2: filter.LowStockOnly,
		Column3: filter.Search,
		Column4: pgtype.Text{String: filter.Sort, Valid: filter.Sort != ""},
		Limit:   int32(filter.Limit),
		Offset:  int32(filter.Offset),
	}

	rows, err := r.queries.GetAllVariantsWithProduct(ctx, arg)
	if err != nil {
		return nil, 0, err
	}

	// Get Count
	count, err := r.queries.CountAllVariantsWithProduct(ctx, sqlc.CountAllVariantsWithProductParams{
		Column1: prodUUID,
		Column2: filter.LowStockOnly,
		Column3: filter.Search,
	})
	if err != nil {
		return nil, 0, err
	}

	variants := make([]domain.VariantWithProduct, len(rows))
	for i, row := range rows {
		// Map Variant fields
		v := domain.Variant{
			ID:                uuidToString(row.ID),
			ProductID:         uuidToString(row.ProductID),
			Name:              row.Name,
			Stock:             int(row.Stock),
			SKU:               ptrStrToStr(row.Sku),
			Price:             numericToFloat64Ptr(row.Price),
			SalePrice:         numericToFloat64Ptr(row.SalePrice),
			Weight:            numericToFloat64Ptr(row.Weight),
			Barcode:           ptrStrToStr(row.Barcode),
			LowStockThreshold: int(row.LowStockThreshold),
		}

		if row.Attributes != nil {
			var attrs domain.JSONB
			if err := json.Unmarshal(row.Attributes, &attrs); err == nil {
				v.Attributes = attrs
			}
		}
		if row.Dimensions != nil {
			var dims domain.JSONB
			if err := json.Unmarshal(row.Dimensions, &dims); err == nil {
				v.Dimensions = dims
			}
		}
		if row.Images != nil {
			v.Images = row.Images
		}

		// Map Product Context
		vp := domain.VariantWithProduct{
			Variant:          v,
			ProductName:      row.ProductName,
			ProductSlug:      row.ProductSlug,
			ProductBasePrice: numericToFloat64(row.ProductBasePrice),
		}

		// Extract first product image if available
		if row.ProductMedia != nil {
			var media struct {
				Images []string `json:"images"`
			}
			if err := json.Unmarshal(row.ProductMedia, &media); err == nil && len(media.Images) > 0 {
				vp.ProductImage = media.Images[0]
			}
		}

		variants[i] = vp
	}

	return variants, count, nil
}

func (r *productRepository) GetProductStats(ctx context.Context) (*domain.ProductStats, error) {
	row, err := r.queries.GetProductStats(ctx)
	if err != nil {
		return nil, err
	}

	return &domain.ProductStats{
		TotalProducts:       row.TotalProducts,
		ActiveProducts:      row.ActiveProducts,
		InactiveProducts:    row.InactiveProducts,
		OutOfStock:          row.OutOfStock,
		LowStock:            row.LowStock,
		TotalInventoryValue: row.TotalInventoryValue,
	}, nil
}

// --- Reviews ---

func (r *productRepository) CreateReview(ctx context.Context, review *domain.Review) error {
	created, err := r.queries.CreateReview(ctx, sqlc.CreateReviewParams{
		ProductID: stringToUUID(review.ProductID),
		UserID:    stringToUUID(review.UserID),
		Rating:    int32(review.Rating),
		Comment:   strPtr(review.Comment),
	})
	if err != nil {
		return err
	}
	review.ID = uuidToString(created.ID)
	review.CreatedAt = pgtimeToTime(created.CreatedAt)
	return nil
}

func (r *productRepository) GetReviews(ctx context.Context, productID string) ([]domain.Review, error) {
	rows, err := r.queries.GetReviewsByProductID(ctx, stringToUUID(productID))
	if err != nil {
		return nil, err
	}

	result := make([]domain.Review, len(rows))
	for i, row := range rows {
		result[i] = domain.Review{
			ID:        uuidToString(row.ID),
			ProductID: uuidToString(row.ProductID),
			UserID:    uuidToString(row.UserID),
			Rating:    int(row.Rating),
			Comment:   ptrString(row.Comment),
			CreatedAt: pgtimeToTime(row.CreatedAt),
			User: domain.User{
				FirstName: ptrString(row.FirstName),
				LastName:  ptrString(row.LastName),
				Avatar:    ptrString(row.Avatar),
			},
		}
	}
	return result, nil
}

// GetVariantByID fetches a single variant by ID
func (r *productRepository) GetVariantByID(ctx context.Context, id string) (*domain.Variant, error) {
	v, err := r.queries.GetVariantByID(ctx, stringToUUID(id))
	if err != nil {
		return nil, err
	}
	variant := sqlcVariantToDomain(v)
	return &variant, nil
}
