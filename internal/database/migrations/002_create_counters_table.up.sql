CREATE TABLE IF NOT EXISTS counters (
                                        id SERIAL PRIMARY KEY,
                                        name VARCHAR(100) NOT NULL UNIQUE,
    value BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_counters_name ON counters(name);
CREATE INDEX IF NOT EXISTS idx_counters_updated_at ON counters(updated_at);