-- name: BalanceTransaction_CreateBatch :batchexec
INSERT INTO balance_transactions
    (user_id, order_id, amount, direction)
VALUES
    ($1, $2, $3, @direction::balance_transaction_type);

-- name: BalanceTransaction_SumByUserID :many
SELECT SUM(amount)::numeric AS sum, direction FROM balance_transactions WHERE user_id = $1 GROUP BY direction;

-- name: BalanceTransaction_CreateByOrderCode :one
INSERT INTO balance_transactions
(user_id, order_id, amount, direction)
VALUES
    ($1, $2, $3, @direction::balance_transaction_type)
RETURNING *;