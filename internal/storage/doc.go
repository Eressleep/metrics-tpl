// Package storage предоставляет интерфейсы и реализации для хранения метрик.
//
// Основные реализации:
//   - MemStorage - хранение метрик в памяти
//   - FileStorage - хранение метрик в файле с периодическим сохранением
//   - DBStorage - хранение метрик в базе данных PostgreSQL
//
// Пример использования:
//
//	store := storage.NewMemStorage()
//	store.UpdateGauge("temperature", 23.5)
//	value, err := store.GetGauge("temperature")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Temperature: %g\n", value)
package storage
