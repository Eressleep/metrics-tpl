// Package server предоставляет HTTP сервер для сбора и хранения метрик.
//
// Сервер поддерживает:
//   - Прием метрик через URL параметры и JSON
//   - Пакетное обновление метрик
//   - Хранение в памяти, файле или базе данных
//   - Подпись запросов с помощью SHA256
//   - Сжатие ответов gzip
//   - Аудит обновлений метрик
//
// Пример запуска сервера:
//
//	config := server.NewDefaultConfig()
//	config.Addr = ":8080"
//
//	store := storage.NewMemStorage()
//	handler := handlers.NewMetricsHandler(store, "secret-key")
//	srv := server.New(config, handler)
//
//	if err := srv.Run(); err != nil {
//	    log.Fatal(err)
//	}
package server
