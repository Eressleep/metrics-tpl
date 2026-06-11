// Package middleware предоставляет промежуточное ПО для HTTP сервера.
//
// Доступные middleware:
//   - GzipMiddleware - сжатие ответов и распаковка запросов
//   - HashCheckMiddleware - проверка хеша SHA256 для запросов
//   - HashResponseMiddleware - добавление хеша SHA256 к ответам
//   - AuditMiddleware - аудит успешных обновлений метрик
//
// Пример использования:
//
//	router := gin.Default()
//	router.Use(middleware.GzipMiddleware())
//	router.Use(middleware.HashCheckMiddleware("secret-key", logger))
//	router.Use(middleware.AuditMiddleware(auditor, logger))
//	router.Run(":8080")
package middleware
