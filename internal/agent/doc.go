// Package agent предоставляет агента для сбора и отправки метрик на сервер.
//
// Агент периодически собирает метрики системы (runtime, память, CPU) и отправляет
// их на сервер метрик. Поддерживает настраиваемые интервалы сбора и отправки,
// а также ограничение частоты запросов (rate limiting).
//
// Пример использования:
//
//	config := agent.DefaultConfig()
//	config.ServerAddr = "localhost:8080"
//	config.PollInterval = 2 * time.Second
//	config.ReportInterval = 10 * time.Second
//
//	agent := agent.New(config)
//	agent.Run()
package agent
