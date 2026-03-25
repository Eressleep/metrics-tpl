CREATE TABLE IF NOT EXISTS gauges (
                                      name TEXT PRIMARY KEY,
                                      value DOUBLE PRECISION NOT NULL,
                                      updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS counters (
                                        name TEXT PRIMARY KEY,
                                        value BIGINT NOT NULL,
                                        updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_gauges_updated_at ON gauges(updated_at);
CREATE INDEX IF NOT EXISTS idx_counters_updated_at ON counters(updated_at);