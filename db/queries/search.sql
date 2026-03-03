-- name: SearchProducts :many
SELECT *,
       COALESCE((ts_rank(search_vector, websearch_to_tsquery('english', @query)) +
        similarity(name, @query) * 2.0 +
        similarity(COALESCE(brand, ''), @query))::float8, 0)::float8 as rank
FROM products
WHERE (
    search_vector @@ websearch_to_tsquery('english', @query)
    OR name ILIKE '%' || @query || '%'
    OR similarity(name, @query) > 0.15
    OR brand ILIKE '%' || @query || '%'
    OR similarity(COALESCE(brand, ''), @query) > 0.15
)
  AND (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'))
ORDER BY rank DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountSearchProducts :one
SELECT COUNT(*)
FROM products
WHERE (
    search_vector @@ websearch_to_tsquery('english', @query)
    OR name ILIKE '%' || @query || '%'
    OR similarity(name, @query) > 0.15
    OR brand ILIKE '%' || @query || '%'
    OR similarity(COALESCE(brand, ''), @query) > 0.15
)
  AND (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'));
