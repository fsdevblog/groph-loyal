CREATE OR REPLACE FUNCTION update_updated_at_column()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Users
CREATE TRIGGER set_updated_at_users
    BEFORE UPDATE ON users
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Orders
CREATE TRIGGER set_updated_at_orders
    BEFORE UPDATE ON orders
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- BalanceTransactions
CREATE TRIGGER set_updated_at_balance_transactions
    BEFORE UPDATE ON balance_transactions
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
