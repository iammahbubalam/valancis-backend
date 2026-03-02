-- name: GetVariantsByProductID :many
SELECT * FROM variants WHERE product_id = $1;

-- name: GetVariantByID :one
SELECT * FROM variants WHERE id = $1;

-- name: CreateVariant :one
INSERT INTO variants (
    product_id, name, stock, sku, 
    attributes, price, sale_price, images, weight, dimensions, barcode, 
    low_stock_threshold
) VALUES (
    $1, $2, $3, $4, 
    $5, $6, $7, $8, $9, $10, $11,
    $12
) RETURNING *;

-- name: UpdateVariant :one
UPDATE variants 
SET name = $2, stock = $3, sku = $4,
    attributes = $5, price = $6, sale_price = $7, 
    images = $8, weight = $9, dimensions = $10, barcode = $11,
    low_stock_threshold = $12
WHERE id = $1 
RETURNING *;

-- name: UpdateVariantStock :execrows
UPDATE variants SET stock = stock + $2 WHERE id = $1 AND stock + $2 >= 0;

-- name: DeleteVariant :exec
DELETE FROM variants WHERE id = $1;

-- name: DeleteVariantsByProductID :exec
DELETE FROM variants WHERE product_id = $1;

-- name: CreateInventoryLog :one
INSERT INTO inventory_logs (product_id, variant_id, change_amount, reason, reference_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetInventoryLogs :many
SELECT * FROM inventory_logs 
WHERE ($1::uuid IS NULL OR product_id = $1)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountInventoryLogs :one
SELECT COUNT(*) FROM inventory_logs WHERE ($1::uuid IS NULL OR product_id = $1);

-- name: GetVariantsByProductIDs :many
SELECT * FROM variants WHERE product_id = ANY($1::uuid[]);

-- name: GetAllVariantsWithProduct :many
SELECT 
    v.id,
    v.product_id,
    v.name,
    v.stock,
    v.sku,
    v.attributes,
    v.price,
    v.sale_price,
    v.images,
    v.weight,
    v.dimensions,
    v.barcode,
    v.low_stock_threshold,
    v.created_at,
    v.updated_at,
    p.name AS product_name,
    p.slug AS product_slug,
    p.base_price AS product_base_price,
    p.media AS product_media
FROM variants v
JOIN products p ON v.product_id = p.id
WHERE 
    ($1::uuid IS NULL OR v.product_id = $1)
    AND ($2::boolean = false OR v.stock <= v.low_stock_threshold)
    AND ($3::text = '' OR v.sku ILIKE '%' || $3 || '%' OR v.name ILIKE '%' || $3 || '%' OR p.name ILIKE '%' || $3 || '%')
ORDER BY 
    CASE WHEN $4 = 'stock_asc' THEN v.stock END ASC,
    CASE WHEN $4 = 'stock_desc' THEN v.stock END DESC,
    CASE WHEN $4 = 'sku_asc' THEN v.sku END ASC,
    CASE WHEN $4 = '' OR $4 IS NULL THEN v.created_at END DESC
LIMIT $5 OFFSET $6;

-- name: CountAllVariantsWithProduct :one
SELECT COUNT(*) FROM variants v
JOIN products p ON v.product_id = p.id
WHERE 
    ($1::uuid IS NULL OR v.product_id = $1)
    AND ($2::boolean = false OR v.stock <= v.low_stock_threshold)
    AND ($3::text = '' OR v.sku ILIKE '%' || $3 || '%' OR v.name ILIKE '%' || $3 || '%' OR p.name ILIKE '%' || $3 || '%');
