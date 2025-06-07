CREATE TYPE balance_transaction_type AS ENUM('debit', 'credit');
CREATE TABLE balance_transactions (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE NOT NULL,
    order_id BIGINT REFERENCES orders(id) ON DELETE CASCADE NOT NULL,
    order_code VARCHAR(20) NOT NULL, -- чтоб избегать join таблиц при выборке, т.к. код всегда нужен везде.
    amount DECIMAL(10,2) NOT NULL DEFAULT 0,
    direction balance_transaction_type NOT NULL
);

CREATE UNIQUE INDEX idx_uniq_order_direction ON balance_transactions(order_id, direction);