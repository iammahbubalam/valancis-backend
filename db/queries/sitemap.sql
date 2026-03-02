-- name: ListProductSlugs :many
SELECT slug, updated_at FROM products WHERE is_active = true ORDER BY updated_at DESC;

-- name: ListCategorySlugs :many
SELECT slug, updated_at FROM categories WHERE is_active = true ORDER BY updated_at DESC;

-- name: ListCollectionSlugs :many
SELECT slug, updated_at FROM collections WHERE is_active = true ORDER BY updated_at DESC;
