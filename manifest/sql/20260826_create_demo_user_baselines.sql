CREATE TABLE IF NOT EXISTS demo_user_baselines (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    user_snapshot JSONB NOT NULL,
    accounts_snapshot JSONB NOT NULL,
    transactions_snapshot JSONB NOT NULL,
    generation_runs_snapshot JSONB NOT NULL,
    last_reset_date DATE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
