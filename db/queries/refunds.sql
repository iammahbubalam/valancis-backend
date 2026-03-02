-- name: CreateRefund :one
INSERT INTO refunds (order_id, amount, reason, restock_items, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateOrderRefundedAmount :exec
UPDATE orders 
SET refunded_amount = refunded_amount + sqlc.arg(amount),
    payment_status = CASE 
        WHEN (refunded_amount + sqlc.arg(amount)) >= paid_amount THEN 'refunded'
        ELSE 'partial_refund'
    END,
    status = CASE 
        WHEN (refunded_amount + sqlc.arg(amount)) >= paid_amount THEN 'refunded'
        ELSE status
    END
WHERE id = $1;

-- name: GetRefundsByOrderID :many
SELECT r.*, u.first_name as processed_by_name
FROM refunds r
LEFT JOIN users u ON u.id = r.created_by
WHERE r.order_id = $1
ORDER BY r.created_at DESC;
