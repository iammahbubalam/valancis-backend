-- name: GetCartByUserID :one
SELECT * FROM carts WHERE user_id = $1;

-- name: CreateCart :one
INSERT INTO carts (user_id) VALUES ($1) RETURNING *;

-- name: GetCartItems :many
SELECT ci.*, p.name, p.slug, p.base_price, p.sale_price, p.media, v.stock, v.sku
FROM cart_items ci
JOIN products p ON p.id = ci.product_id
JOIN variants v ON v.id = ci.variant_id
WHERE ci.cart_id = $1;

-- name: GetCartWithItems :many
SELECT 
    c.id as cart_id,
    c.user_id,
    ci.id as item_id,
    ci.product_id,
    ci.variant_id,
    ci.quantity,
    p.name,
    p.slug,
    p.base_price,
    p.sale_price,
    p.media,
    p.stock_status,
    v.stock,
    v.sku as variant_sku,
    v.name as variant_name,
    v.images as variant_images,
    v.price as variant_price,
    v.sale_price as variant_sale_price
FROM carts c
LEFT JOIN cart_items ci ON c.id = ci.cart_id
LEFT JOIN products p ON ci.product_id = p.id
LEFT JOIN variants v ON ci.variant_id = v.id
WHERE c.user_id = $1;


-- name: GetCartItemByProductID :one
SELECT * FROM cart_items WHERE cart_id = $1 AND product_id = $2;

-- name: UpsertCartItemAtomic :many
-- L9 FIX: Simplified atomic upsert without expression-based conflict target
WITH 
  user_cart AS (
    SELECT c.id FROM carts c
    WHERE c.id = sqlc.arg(cart_id) AND c.user_id = sqlc.arg(user_id)
  ),
  stock_valid AS (
    SELECT v.id FROM variants v
    JOIN products p ON p.id = v.product_id
    WHERE v.id = sqlc.arg(variant_id)
      AND (v.stock >= sqlc.arg(quantity) OR p.stock_status = 'pre_order')
      AND p.is_active = TRUE
  ),
  existing_item AS (
    SELECT ci.id FROM cart_items ci
    WHERE ci.cart_id = sqlc.arg(cart_id)
      AND ci.product_id = sqlc.arg(product_id)
      AND (ci.variant_id IS NOT DISTINCT FROM sqlc.arg(variant_id))
  ),
  updated AS (
    UPDATE cart_items SET quantity = sqlc.arg(quantity)
    WHERE id = (SELECT id FROM existing_item)
      AND EXISTS (SELECT 1 FROM stock_valid)
    RETURNING id, cart_id, product_id, variant_id, quantity
  ),
  inserted AS (
    INSERT INTO cart_items (cart_id, product_id, variant_id, quantity)
    SELECT uc.id, sqlc.arg(product_id), sqlc.arg(variant_id), sqlc.arg(quantity)
    FROM user_cart uc
    CROSS JOIN stock_valid sv
    WHERE NOT EXISTS (SELECT 1 FROM existing_item)
    RETURNING id, cart_id, product_id, variant_id, quantity
  ),
  results AS (
    SELECT * FROM updated
    UNION ALL
    SELECT * FROM inserted
  )
SELECT r.id, r.cart_id, r.product_id, r.variant_id, r.quantity,
       p.name, p.slug, p.base_price, p.sale_price, p.media, v.stock, v.sku as variant_sku, v.name as variant_name, v.images as variant_images,
       v.price as variant_price, v.sale_price as variant_sale_price
FROM results r
JOIN products p ON p.id = r.product_id
JOIN variants v ON v.id = r.variant_id;




-- name: AtomicRemoveCartItem :exec
DELETE FROM cart_items ci
USING carts c
WHERE ci.cart_id = c.id
  AND c.user_id = $1
  AND ci.product_id = $2
  AND ci.variant_id = $3;

-- name: ClearCart :exec
DELETE FROM cart_items WHERE cart_id = $1;

-- name: CreateOrder :one
INSERT INTO orders (user_id, status, total_amount, shipping_fee, shipping_address, payment_method, payment_status, paid_amount, payment_details, is_preorder)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetOrderByID :one
SELECT * FROM orders WHERE id = $1;

-- name: GetOrdersByUserID :many
SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC;

-- name: GetAllOrders :many
SELECT o.*, u.email, u.first_name, u.last_name
FROM orders o
JOIN users u ON u.id = o.user_id
WHERE 
    (sqlc.narg('status')::text IS NULL OR o.status = sqlc.narg('status')) AND
    (sqlc.narg('payment_status')::text IS NULL OR o.payment_status = sqlc.narg('payment_status')) AND
    (sqlc.narg('is_preorder')::boolean IS NULL OR o.is_preorder = sqlc.narg('is_preorder')) AND
    (sqlc.narg('search')::text IS NULL OR 
        o.id::text ILIKE '%' || sqlc.narg('search') || '%' OR 
        u.email ILIKE '%' || sqlc.narg('search') || '%' OR 
        o.payment_details->>'transaction_id' ILIKE '%' || sqlc.narg('search') || '%' OR
        o.payment_details->>'sender_number' ILIKE '%' || sqlc.narg('search') || '%'
    )
ORDER BY o.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountOrders :one
SELECT COUNT(*) 
FROM orders o
JOIN users u ON u.id = o.user_id
WHERE 
    (sqlc.narg('status')::text IS NULL OR o.status = sqlc.narg('status')) AND
    (sqlc.narg('payment_status')::text IS NULL OR o.payment_status = sqlc.narg('payment_status')) AND
    (sqlc.narg('is_preorder')::boolean IS NULL OR o.is_preorder = sqlc.narg('is_preorder')) AND
    (sqlc.narg('search')::text IS NULL OR 
        o.id::text ILIKE '%' || sqlc.narg('search') || '%' OR 
        u.email ILIKE '%' || sqlc.narg('search') || '%' OR 
        o.payment_details->>'transaction_id' ILIKE '%' || sqlc.narg('search') || '%' OR
        o.payment_details->>'sender_number' ILIKE '%' || sqlc.narg('search') || '%'
    );

-- name: UpdateOrderStatus :exec
UPDATE orders SET status = $2 WHERE id = $1;

-- name: UpdateOrderPaymentStatus :exec
UPDATE orders SET payment_status = $2 WHERE id = $1;

-- name: CreateOrderItem :one
INSERT INTO order_items (order_id, product_id, variant_id, quantity, price)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetOrderItems :many
SELECT oi.*, p.name, p.slug, p.media, v.name as variant_name, v.sku as variant_sku
FROM order_items oi
JOIN products p ON p.id = oi.product_id
LEFT JOIN variants v ON v.id = oi.variant_id
WHERE oi.order_id = $1;

-- name: HasPurchasedProduct :one
SELECT EXISTS (
    SELECT 1
    FROM order_items oi
    JOIN orders o ON o.id = oi.order_id
    WHERE o.user_id = $1 
      AND oi.product_id = $2
      AND o.status = 'delivered'
);

-- name: CreateOrderHistory :one
INSERT INTO order_history (order_id, previous_status, new_status, reason, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetOrderHistory :many
SELECT oh.*, u.first_name, u.last_name, u.email
FROM order_history oh
LEFT JOIN users u ON u.id = oh.created_by
WHERE oh.order_id = $1
ORDER BY oh.created_at DESC;
