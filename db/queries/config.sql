-- name: GetActiveShippingZones :many
SELECT * FROM shipping_zones WHERE is_active = TRUE ORDER BY id ASC;

-- name: GetAllShippingZones :many
SELECT * FROM shipping_zones ORDER BY id ASC;

-- name: GetShippingZoneByID :one
SELECT * FROM shipping_zones WHERE id = $1;

-- name: GetShippingZoneByKey :one
SELECT * FROM shipping_zones WHERE key = $1 AND is_active = TRUE;

-- name: CreateShippingZone :one
INSERT INTO shipping_zones (key, label, cost, is_active)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateShippingZone :one
UPDATE shipping_zones
SET label = $2, cost = $3, is_active = $4, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateShippingZoneCost :exec
UPDATE shipping_zones SET cost = $2, updated_at = CURRENT_TIMESTAMP WHERE key = $1;

-- name: DeleteShippingZone :exec
DELETE FROM shipping_zones WHERE id = $1;
