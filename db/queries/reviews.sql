-- name: CreateReview :one
INSERT INTO reviews (product_id, user_id, rating, comment)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetReviewsByProductID :many
SELECT r.*, u.first_name, u.last_name, u.avatar 
FROM reviews r
JOIN users u ON u.id = r.user_id
WHERE r.product_id = $1
ORDER BY r.created_at DESC;

-- name: GetReviewByID :one
SELECT * FROM reviews WHERE id = $1;

-- name: DeleteReview :exec
DELETE FROM reviews WHERE id = $1;

-- name: GetUserReviewForProduct :one
SELECT * FROM reviews WHERE product_id = $1 AND user_id = $2;
