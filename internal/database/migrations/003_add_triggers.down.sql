DROP TRIGGER IF EXISTS update_gauges_updated_at ON gauges;
DROP TRIGGER IF EXISTS update_counters_updated_at ON counters;
DROP FUNCTION IF EXISTS update_updated_at_column();