-- name: GetActiveCollections :many
SELECT * FROM collections WHERE is_active = true ORDER BY created_at DESC;

-- name: GetAllCollections :many
SELECT * FROM collections ORDER BY created_at DESC;

-- name: GetCollectionBySlug :one
SELECT * FROM collections WHERE slug = $1;

-- name: GetCollectionByID :one
SELECT * FROM collections WHERE id = $1;

-- name: CreateCollection :one
INSERT INTO collections (title, slug, description, image, story, is_active, meta_title, meta_description, meta_keywords, og_image)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateCollection :one
UPDATE collections
SET title = $2, slug = $3, description = $4, image = $5, story = $6, is_active = $7,
    meta_title = $8, meta_description = $9, meta_keywords = $10, og_image = $11
WHERE id = $1
RETURNING *;

-- name: DeleteCollection :exec
DELETE FROM collections WHERE id = $1;

-- name: AddProductToCollection :exec
INSERT INTO product_collections (product_id, collection_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveProductFromCollection :exec
DELETE FROM product_collections WHERE product_id = $1 AND collection_id = $2;

-- name: GetProductIDsForCollection :many
SELECT product_id FROM product_collections WHERE collection_id = $1;
