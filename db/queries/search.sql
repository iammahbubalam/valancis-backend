-- name: SearchProducts :many
SELECT *,
       ts_rank(search_vector, websearch_to_tsquery('english', @query)) as rank
FROM products
WHERE search_vector @@ websearch_to_tsquery('english', @query)
  AND (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'))
ORDER BY rank DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountSearchProducts :one
SELECT COUNT(*)
FROM products
WHERE search_vector @@ websearch_to_tsquery('english', @query)
  AND (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'));
