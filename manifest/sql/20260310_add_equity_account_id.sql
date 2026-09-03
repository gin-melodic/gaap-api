-- Add equity_account_id column to accounts table
-- Links source accounts (Asset/Liability) with their Equity counterpart bidirectionally
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS equity_account_id UUID REFERENCES accounts(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_accounts_equity_account_id ON accounts(equity_account_id);
