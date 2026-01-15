-- GAAP Web Database Schema
-- Updated to support GORM soft delete (deleted_at)

-- Drop tables to ensure clean state in dev
SET client_min_messages = warning;
-- DROP TABLE IF EXISTS transactions CASCADE;
-- DROP TABLE IF EXISTS accounts CASCADE;
-- DROP TABLE IF EXISTS account_types CASCADE;
-- DROP TABLE IF EXISTS oauth_connections CASCADE;
-- DROP TABLE IF EXISTS users CASCADE;
-- DROP TABLE IF EXISTS themes CASCADE;
-- DROP TABLE IF EXISTS currencies CASCADE;

-- Disable Foreign Keys Check
SET session_replication_role = replica;

-- -- UUIDv7 Function Definition
-- Postgres 17+ is already supported vanila uuidv7, so we can skip this
-- CREATE OR REPLACE FUNCTION uuidv7() RETURNS uuid AS $$
-- BEGIN
--     -- Generates a UUIDv7 compatible text and casts to UUID
--     -- This is a polyfill for PG < 17
--     return encode(
--       decode(
--         lpad(to_hex(floor(extract(epoch from clock_timestamp()) * 1000)::bigint), 12, '0') ||
--         to_hex(floor(random() * 4294967295)::bigint) || -- 32 bits random
--         to_hex(floor(random() * 4294967295)::bigint),   -- 32 bits random (total 74 random bits needed, here simplistic)
--         'hex'
--       ),
--       'hex'
--     )::uuid;
-- END
-- $$ LANGUAGE plpgsql;

-- Themes Table
CREATE TABLE IF NOT EXISTS themes (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(50) NOT NULL,
    is_dark BOOLEAN NOT NULL DEFAULT FALSE,
    colors JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Currencies Table
CREATE TABLE IF NOT EXISTS currencies (
    code VARCHAR(10) PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Users Table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    "password" VARCHAR(255) NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    nickname VARCHAR(50) NOT NULL,
    avatar VARCHAR(2048),
    plan INTEGER NOT NULL,
    theme_id UUID REFERENCES themes(id) ON DELETE SET NULL,
    main_currency VARCHAR(10) REFERENCES currencies(code) ON DELETE SET NULL,
    two_factor_secret VARCHAR(100),
    two_factor_enabled BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- OAuth Connections Table
CREATE TABLE IF NOT EXISTS oauth_connections (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    "provider" VARCHAR(20) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    access_token VARCHAR(1024),
    refresh_token VARCHAR(1024),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE("provider", provider_user_id)
);

-- Account Types Table
-- Stores configuration for account types (ASSET, LIABILITY, etc.)
CREATE TABLE IF NOT EXISTS account_types (
    "type" INTEGER PRIMARY KEY,
    label VARCHAR(50) NOT NULL,
    color VARCHAR(50),
    bg VARCHAR(50),
    icon VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Accounts Table
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    type INTEGER NOT NULL REFERENCES account_types(type),
    is_group BOOLEAN NOT NULL DEFAULT FALSE,
    currency_code VARCHAR(10) NOT NULL,
    balance_units BIGINT NOT NULL DEFAULT 0,
    balance_nanos INTEGER NOT NULL DEFAULT 0,
    balance_decimal NUMERIC(20, 9) GENERATED ALWAYS AS ( balance_units + (balance_nanos::NUMERIC / 1000000000) ) STORED,
    default_child_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    date DATE,
    number VARCHAR(50),
    remarks VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Transactions Table
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    from_account_id UUID NOT NULL REFERENCES accounts(id),
    to_account_id UUID NOT NULL REFERENCES accounts(id),
    currency_code VARCHAR(10) NOT NULL,
    balance_units BIGINT NOT NULL DEFAULT 0,
    balance_nanos INTEGER NOT NULL DEFAULT 0,
    balance_decimal NUMERIC(20, 9) GENERATED ALWAYS AS ( balance_units + (balance_nanos::NUMERIC / 1000000000) ) STORED,
    note VARCHAR(500),
    type INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Tasks Table for async job tracking
CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type INTEGER NOT NULL DEFAULT 0,
    status INTEGER NOT NULL DEFAULT 1,
    payload JSONB NOT NULL,
    result JSONB,
    progress INT DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    total_items INT,
    processed_items INT DEFAULT 0,                                  -- Items processed so far
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Migration Mappings Table for safe transaction migration
CREATE TABLE IF NOT EXISTS migration_mappings (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    table_name VARCHAR(50) NOT NULL,                                -- 'transactions'
    record_id UUID NOT NULL,
    field_name VARCHAR(50) NOT NULL,                                -- 'from_account_id' or 'to_account_id'
    old_value UUID NOT NULL,
    new_value UUID NOT NULL,
    applied BOOLEAN DEFAULT FALSE,                                  -- Whether the change has been applied
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_accounts_parent_id ON accounts(parent_id);
CREATE INDEX IF NOT EXISTS idx_accounts_type ON accounts(type);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
CREATE INDEX IF NOT EXISTS idx_transactions_from_account ON transactions(from_account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_to_account ON transactions(to_account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);

-- Indexes for tasks
CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(type);
CREATE INDEX IF NOT EXISTS idx_migration_mappings_task_id ON migration_mappings(task_id);
CREATE INDEX IF NOT EXISTS idx_migration_mappings_record_id ON migration_mappings(record_id);

-- Indexes for soft deletes (GORM compatibility)
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_themes_deleted_at ON themes(deleted_at);
CREATE INDEX IF NOT EXISTS idx_currencies_deleted_at ON currencies(deleted_at);
CREATE INDEX IF NOT EXISTS idx_account_types_deleted_at ON account_types(deleted_at);
CREATE INDEX IF NOT EXISTS idx_accounts_deleted_at ON accounts(deleted_at);
CREATE INDEX IF NOT EXISTS idx_transactions_deleted_at ON transactions(deleted_at);

-- Indexes for user_id
CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);

-- Comments
COMMENT ON TABLE users IS 'Stores user profiles';
COMMENT ON TABLE themes IS 'Stores user interface themes';
COMMENT ON TABLE currencies IS 'Supported currencies';
COMMENT ON TABLE account_types IS 'Configuration for different account types';
COMMENT ON TABLE accounts IS 'Financial accounts hierarchy';
COMMENT ON TABLE transactions IS 'Financial transactions between accounts';
COMMENT ON TABLE tasks IS 'Async task tracking for long-running operations';
COMMENT ON TABLE migration_mappings IS 'Temporary storage for safe data migration';

-- Resume Foreign Keys Check
SET session_replication_role = DEFAULT;

