CREATE TABLE IF NOT EXISTS gauges (
                                      id SERIAL PRIMARY KEY,
                                      name VARCHAR(100) NOT NULL UNIQUE,
    value DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_gauges_name ON gauges(name);
CREATE INDEX IF NOT EXISTS idx_gauges_updated_at ON gauges(updated_at);