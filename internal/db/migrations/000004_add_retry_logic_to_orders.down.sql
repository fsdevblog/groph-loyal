DROP INDEX idx_orders_monitoring;
ALTER TABLE orders DROP COLUMN attempts, DROP column next_attempt_at;