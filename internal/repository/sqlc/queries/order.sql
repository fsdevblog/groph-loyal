-- name: Orders_Create :one
INSERT INTO orders
    (user_id, order_code, status, accrual)
VALUES
    ($1, $2, $3, $4)
RETURNING *;

-- name: Orders_FindByOrderCode :one
SELECT * FROM orders WHERE order_code = $1;