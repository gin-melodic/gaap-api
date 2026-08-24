-- Dashboard Snapshots Table
-- Persists dashboard snapshot data to survive Redis restarts / memory pressure.
-- Snapshots are flushed from Redis to DB at configurable intervals (default: daily).
CREATE TABLE IF NOT EXISTS dashboard_snapshots (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    snapshot_type VARCHAR(20) NOT NULL,  -- 'summary', 'monthly', 'trend'
    snapshot_key VARCHAR(100) NOT NULL,  -- month key for monthly (e.g. '2026-02'), empty for others
    data JSONB NOT NULL,                 -- serialised snapshot payload
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, snapshot_type, snapshot_key)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_dashboard_snapshots_user_id ON dashboard_snapshots(user_id);
CREATE INDEX IF NOT EXISTS idx_dashboard_snapshots_type ON dashboard_snapshots(snapshot_type);

COMMENT ON TABLE dashboard_snapshots IS 'Persisted dashboard snapshot data for cold-start recovery';
