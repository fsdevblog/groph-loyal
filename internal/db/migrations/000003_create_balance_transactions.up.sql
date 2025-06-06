CREATE TYPE balance_transaction_type AS ENUM('debit', 'credit');
CREATE TABLE balance_transactions (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE NOT NULL,
    order_id BIGINT REFERENCES orders(id) ON DELETE CASCADE DEFAULT NULL,
    amount DECIMAL(10,2) NOT NULL DEFAULT 0,
    direction balance_transaction_type NOT NULL
);

CREATE UNIQUE INDEX idx_uniq_order ON balance_transactions(order_id);