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

-- name: Orders_GetForMonitoring :many
SELECT * FROM orders
WHERE status IN ('NEW', 'PROCESSING')
  AND (next_attempt_at IS NULL OR
       next_attempt_at <= NOW())
ORDER BY attempts, created_at
LIMIT @_limit;


-- name: Orders_IncrementAttempts :batchexec
UPDATE orders
SET attempts = attempts + 1,
    next_attempt_at = $1,
    updated_at = NOW()
WHERE id = $2;

-- name: Orders_UpdateWithAccrualData :batchone
UPDATE orders SET status = $1, accrual = $2, updated_at = NOW() WHERE id = $3 RETURNING *;