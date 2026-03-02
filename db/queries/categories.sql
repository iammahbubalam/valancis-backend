-- name: GetAllCategories :many
SELECT * FROM categories ORDER BY order_index ASC;

-- name: GetRootCategories :many
SELECT * FROM categories WHERE parent_id IS NULL ORDER BY order_index ASC;

-- name: GetActiveNavCategories :many
SELECT * FROM categories 
WHERE parent_id IS NULL AND is_active = true AND show_in_nav = true 
ORDER BY order_index ASC;

-- name: GetChildCategories :many
SELECT * FROM categories WHERE parent_id = $1 ORDER BY order_index ASC;

-- name: GetActiveChildCategories :many
SELECT * FROM categories 
WHERE parent_id = $1 AND is_active = true AND show_in_nav = true 
ORDER BY order_index ASC;

-- name: GetCategoryBySlug :one
SELECT * FROM categories WHERE slug = $1;

-- name: GetCategoryByID :one
SELECT * FROM categories WHERE id = $1;

-- name: CreateCategory :one
INSERT INTO categories (name, slug, parent_id, order_index, icon, image, is_active, show_in_nav, meta_title, meta_description, keywords, is_featured, og_image)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: UpdateCategory :one
UPDATE categories
SET name = $2, slug = $3, parent_id = $4, order_index = $5, icon = $6, image = $7, 
    is_active = $8, show_in_nav = $9, meta_title = $10, meta_description = $11, keywords = $12, is_featured = $13, og_image = $14
WHERE id = $1
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = $1;

-- name: UpdateCategoryOrder :exec
UPDATE categories SET order_index = $2, parent_id = $3 WHERE id = $1;

-- name: GetCategoriesByIDs :many
SELECT * FROM categories WHERE id = ANY($1::uuid[]);

-- name: GetCategoriesFlat :many
SELECT * FROM categories 
WHERE (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'))
ORDER BY name ASC;
