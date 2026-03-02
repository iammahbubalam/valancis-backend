-- name: CreateUser :one
INSERT INTO users (email, role, first_name, last_name, avatar)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUser :one
UPDATE users
SET email = $2, role = $3, first_name = $4, last_name = $5, avatar = $6
WHERE id = $1
RETURNING *;

-- name: CreateAddress :one
INSERT INTO addresses (user_id, label, contact_email, phone, first_name, last_name, delivery_zone, division, district, thana, address_line, landmark, postal_code, is_default)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: GetAddressesByUserID :many
SELECT * FROM addresses WHERE user_id = $1 ORDER BY created_at DESC;

-- name: UpdateAddress :one
UPDATE addresses
SET label = $2, contact_email = $3, phone = $4, first_name = $5, last_name = $6, delivery_zone = $7, division = $8, district = $9, thana = $10, address_line = $11, landmark = $12, postal_code = $13, is_default = $14
WHERE id = $1 AND user_id = $15
RETURNING *;

-- name: DeleteAddress :exec
DELETE FROM addresses WHERE id = $1 AND user_id = $2;

-- name: UpdateUserProfile :one
UPDATE users
SET first_name = $2, last_name = $3, phone = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SaveRefreshToken :one
INSERT INTO refresh_tokens (token, user_id, expires_at, device)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens WHERE token = $1 AND revoked = false;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked = true WHERE token = $1;

