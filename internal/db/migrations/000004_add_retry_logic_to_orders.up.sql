ALTER TABLE orders
    ADD COLUMN attempts INT DEFAULT 0 NOT NULL,
    ADD COLUMN next_attempt_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;

CREATE INDEX idx_orders_monitoring
    ON orders
        (status, next_attempt_at, attempts, created_at)
    WHERE status IN ('NEW', 'PROCESSING');


