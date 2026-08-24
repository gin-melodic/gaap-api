CREATE TABLE IF NOT EXISTS demo_data_generation_runs (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    business_date DATE NOT NULL,
    generated_count INTEGER NOT NULL DEFAULT 0 CHECK (generated_count >= 0),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, business_date)
);

