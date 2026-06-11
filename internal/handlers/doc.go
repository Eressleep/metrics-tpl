// Package handlers предоставляет HTTP обработчики для сервера метрик.
//
// MetricsHandler обрабатывает запросы на обновление и получение метрик
// в различных форматах (текстовый URL и JSON).
//
// Поддерживаемые эндпоинты:
//   - POST /update/:type/:name/:value - обновление метрики через URL
//   - POST /update/ - обновление метрики через JSON
//   - POST /updates/ - пакетное обновление метрик
//   - POST /value/ - получение значения метрики через JSON
//   - GET /value/:type/:name - получение значения метрики
//   - GET / - просмотр всех метрик в HTML
//   - GET /ping - проверка доступности сервера
//
// Пример обновления метрики:
//
//	handler := handlers.NewMetricsHandler(store, "")
//
//	router := gin.Default()
//	router.POST("/update/:type/:name/:value", handler.Update)
//	router.POST("/update/", handler.UpdateJSON)
//	router.POST("/value/", handler.GetValueJSON)
//	router.GET("/", handler.GetAllMetrics)
//	router.GET("/ping", handler.Ping)
//	router.Run(":8080")
package handlers
