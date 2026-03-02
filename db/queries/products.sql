-- name: GetProducts :many
SELECT * FROM products 
WHERE (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'))
AND (sqlc.narg('is_featured')::boolean IS NULL OR is_featured = sqlc.narg('is_featured'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountProducts :one
SELECT COUNT(*) FROM products 
WHERE (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'))
AND (sqlc.narg('is_featured')::boolean IS NULL OR is_featured = sqlc.narg('is_featured'));

-- name: GetProductBySlug :one
SELECT * FROM products WHERE slug = $1;

-- name: GetProductByID :one
SELECT * FROM products WHERE id = $1;

-- name: CreateProduct :one
INSERT INTO products (
    name, slug, description, base_price, sale_price, 
    stock_status, is_featured, is_active, 
    media, attributes, specifications, 
    meta_title, meta_description, meta_keywords, og_image,
    brand, tags, warranty_info
) VALUES (
    $1, $2, $3, $4, $5, 
    $6, $7, $8, 
    $9, $10, $11, 
    $12, $13, $14, $15,
    $16, $17, $18
) RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET name = $2, slug = $3, description = $4, base_price = $5, sale_price = $6, 
    stock_status = $7, is_featured = $8, 
    is_active = $9, media = $10, attributes = $11, specifications = $12,
    meta_title = $13, meta_description = $14, meta_keywords = $15, og_image = $16,
    brand = $17, tags = $18, warranty_info = $19
WHERE id = $1
RETURNING *;

-- name: UpdateProductStatus :exec
UPDATE products SET is_active = $2 WHERE id = $1;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = $1;


-- name: GetProductsWithCategoryFilter :many
SELECT DISTINCT p.* FROM products p
JOIN product_categories pc ON pc.product_id = p.id
JOIN categories c ON c.id = pc.category_id
WHERE c.slug = sqlc.arg('slug') 
AND (sqlc.narg('is_active')::boolean IS NULL OR p.is_active = sqlc.narg('is_active'))
ORDER BY p.created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountProductsWithCategoryFilter :one
SELECT COUNT(DISTINCT p.id) FROM products p
JOIN product_categories pc ON pc.product_id = p.id
JOIN categories c ON c.id = pc.category_id
WHERE c.slug = sqlc.arg('slug') 
AND (sqlc.narg('is_active')::boolean IS NULL OR p.is_active = sqlc.narg('is_active'));



-- name: GetProductsWithPriceRange :many
SELECT * FROM products
WHERE base_price >= $1 AND base_price <= $2 AND ($3::boolean IS NULL OR is_active = $3)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: AddProductCategory :exec
INSERT INTO product_categories (product_id, category_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveProductCategory :exec
DELETE FROM product_categories WHERE product_id = $1 AND category_id = $2;

-- name: ClearProductCategories :exec
DELETE FROM product_categories WHERE product_id = $1;

-- name: GetCategoryIDsForProduct :many
SELECT category_id FROM product_categories WHERE product_id = $1;

-- name: GetProductsForCollection :many
SELECT p.* FROM products p
JOIN product_collections pc ON pc.product_id = p.id
WHERE pc.collection_id = $1 AND p.is_active = true
ORDER BY p.created_at DESC;

-- name: AddProductCollection :exec
INSERT INTO product_collections (product_id, collection_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveProductCollection :exec
DELETE FROM product_collections WHERE product_id = $1 AND collection_id = $2;

-- name: ClearProductCollections :exec
DELETE FROM product_collections WHERE product_id = $1;

-- name: GetCollectionIDsForProduct :many
SELECT collection_id FROM product_collections WHERE product_id = $1;

-- name: GetCollectionsByIDs :many
SELECT * FROM collections WHERE id = ANY($1::uuid[]);

-- name: GetCategoryIDsForProducts :many
SELECT product_id, category_id FROM product_categories WHERE product_id = ANY($1::uuid[]);

-- name: GetCollectionIDsForProducts :many
SELECT product_id, collection_id FROM product_collections WHERE product_id = ANY($1::uuid[]);

-- name: GetProductStats :one
SELECT 
    COUNT(DISTINCT p.id) as total_products,
    COUNT(DISTINCT p.id) FILTER (WHERE p.is_active = true) as active_products,
    COUNT(DISTINCT p.id) FILTER (WHERE p.is_active = false) as inactive_products,
    COUNT(DISTINCT v.id) FILTER (WHERE v.stock = 0) as out_of_stock,
    COUNT(DISTINCT v.id) FILTER (WHERE v.stock > 0 AND v.stock <= v.low_stock_threshold) as low_stock,
    COALESCE(SUM(COALESCE(v.price, p.base_price) * v.stock), 0)::float8 as total_inventory_value
FROM products p
LEFT JOIN variants v ON v.product_id = p.id;
