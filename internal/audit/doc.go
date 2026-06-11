// Package audit предоставляет функциональность для аудита обновлений метрик.
//
// Auditor поддерживает два типа приемников:
//   - Файловый: запись событий в файл
//   - URL: отправка событий на удаленный сервер
//
// Формат события аудита:
//
//	{
//	    "ts": 1234567890,
//	    "metrics": ["Alloc", "Frees"],
//	    "ip_address": "192.168.0.1"
//	}
//
// Пример использования:
//
//	auditor := audit.New("/var/log/audit.log", "http://audit-server:8080/audit", logger)
//	event := audit.Event{
//	    TS:        time.Now().Unix(),
//	    Metrics:   []string{"Alloc", "Frees"},
//	    IPAddress: "192.168.1.1",
//	}
//	if err := auditor.LogEvent(event); err != nil {
//	    log.Printf("Failed to log audit event: %v", err)
//	}
package audit
