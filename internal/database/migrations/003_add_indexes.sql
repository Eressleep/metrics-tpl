-- +migrate Up
-- Создаем составные индексы для оптимизации запросов
CREATE INDEX IF NOT EXISTS idx_gauges_name_value ON gauges(name, value);
CREATE INDEX IF NOT EXISTS idx_counters_name_value ON counters(name, value);

-- Добавляем триггер для автоматического обновления updated_at
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

-- +migrate Down
DROP TRIGGER IF EXISTS update_gauges_updated_at ON gauges;
DROP TRIGGER IF EXISTS update_counters_updated_at ON counters;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP INDEX IF EXISTS idx_gauges_name_value;
DROP INDEX IF EXISTS idx_counters_name_value;