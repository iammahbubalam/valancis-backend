-- name: UpsertDailySalesStat :exec
INSERT INTO daily_sales_stats (date, total_revenue, total_orders, total_items_sold, avg_order_value)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (date) DO UPDATE
SET total_revenue = EXCLUDED.total_revenue,
    total_orders = EXCLUDED.total_orders,
    total_items_sold = EXCLUDED.total_items_sold,
    avg_order_value = EXCLUDED.avg_order_value,
    updated_at = NOW();

-- name: GetDailySalesStats :many
SELECT * FROM daily_sales_stats
WHERE date >= $1 AND date <= $2
ORDER BY date ASC
LIMIT $3 OFFSET $4;

-- name: GetTotalRevenue :one
SELECT COALESCE(SUM(total_revenue), 0)::numeric FROM daily_sales_stats;
