-- name: Orders_Create :one
INSERT INTO orders
    (user_id, order_code, status, accrual)
VALUES
    ($1, $2, $3, $4)
RETURNING *;

-- name: Orders_FindByOrderCode :one
SELECT * FROM orders WHERE order_code = $1;

-- name: Orders_GetByUserID :many
SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC;

-- name: Orders_GetByStatuses :many
SELECT * FROM orders WHERE status =ANY(@statuses::order_status_type[]) LIMIT @_limit;

-- name: Orders_UpdateWithAccrualData :batchone
UPDATE orders SET status = $1, accrual = $2 WHERE id = $3 RETURNING *;