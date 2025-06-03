CREATE TYPE order_status_type AS ENUM('PROCESSED', 'PROCESSING', 'INVALID', 'REGISTERED', 'NEW');
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE NOT NULL,
    order_code VARCHAR(20) NOT NULL,
    status order_status_type NOT NULL DEFAULT 'NEW',
    accrual INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_uniq_order_code ON orders(order_code);