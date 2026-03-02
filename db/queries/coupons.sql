-- name: CreateCoupon :one
INSERT INTO coupons (code, type, value, min_spend, usage_limit, start_at, expires_at, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetCouponByCode :one
SELECT * FROM coupons 
WHERE code = $1;

-- name: ValidateCoupon :one
-- L9 Optimization: Single-pass validation logic pushed to DB.
-- Returns the coupon if valid, or a status reason if not.
-- Uses covering indexes on (code) and partial indexes on (is_active) where applicable.
SELECT 
    id, 
    code, 
    type, 
    value, 
    min_spend,
    CASE 
        WHEN NOT is_active THEN 'inactive'
        WHEN start_at > NOW() THEN 'not_started'
        WHEN expires_at < NOW() THEN 'expired'
        WHEN usage_limit > 0 AND used_count >= usage_limit THEN 'fully_redeemed'
        WHEN min_spend > @cart_total::decimal THEN 'min_spend_not_met'
        ELSE 'valid'
    END::text as validation_status
FROM coupons 
WHERE code = @code;

-- name: ListCoupons :many
SELECT * FROM coupons 
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: IncrementCouponUsage :exec
-- L9 Optimization: Atomic increment with optimistic concurrency check if needed.
-- We rely on db-level atomicity here.
UPDATE coupons 
SET used_count = used_count + 1 
WHERE id = $1 AND (usage_limit = 0 OR used_count < usage_limit);

-- name: DeleteCoupon :exec
DELETE FROM coupons WHERE id = $1;

-- name: GetCouponByID :one
SELECT * FROM coupons WHERE id = $1;

-- name: UpdateCoupon :exec
UPDATE coupons SET 
    code = $2,
    type = $3,
    value = $4,
    min_spend = $5,
    usage_limit = $6,
    start_at = $7,
    expires_at = $8,
    is_active = $9
WHERE id = $1;

-- name: CountCoupons :one
SELECT COUNT(*) FROM coupons;
