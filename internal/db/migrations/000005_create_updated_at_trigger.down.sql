DROP TRIGGER IF EXISTS set_updated_at_users ON users;
DROP TRIGGER IF EXISTS set_updated_at_orders ON orders;
DROP TRIGGER IF EXISTS set_updated_at_balance_transactions ON balance_transactions;
DROP FUNCTION IF EXISTS update_updated_at_column();