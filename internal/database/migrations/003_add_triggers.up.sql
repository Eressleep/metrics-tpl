CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_gauges_updated_at ON gauges;
CREATE TRIGGER update_gauges_updated_at
    BEFORE UPDATE ON gauges
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_counters_updated_at ON counters;
CREATE TRIGGER update_counters_updated_at
    BEFORE UPDATE ON counters
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();