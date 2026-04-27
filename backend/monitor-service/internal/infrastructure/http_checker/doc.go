// Package http_checker предоставляет HTTP/HTTPS клиент для выполнения проверок
// доступности сервисов.
//
// Пакет реализует HTTP клиента с поддержкой:
//   - Настраиваемого timeout
//   - Отслеживания redirect
//   - Измерения response time
//   - Поддержки различных HTTP методов
//   - Обработки ошибок соединения
//
// Основные компоненты:
//   - Checker: Интерфейс для выполнения проверок
//   - HTTPChecker: Реализация Checker для HTTP/HTTPS
//   - Result: Результат проверки со всеми метриками
//
// Использование:
//
//	checker := http_checker.NewHTTPChecker(&http_checker.Config{
//	    Timeout: 30 * time.Second,
//	})
//
//	result := checker.Check(ctx, "https://api.example.com/health")
//	if result.Error != nil {
//	    log.Printf("check failed: %v", result.Error)
//	    return
//	}
//
//	log.Printf("status: %d, time: %dms", result.StatusCode, result.ResponseTime)
//
// Метрики:
//
//	ResponseTime - Время ответа в миллисекундах
//	StatusCode   - HTTP статус код
//	Success      - true если 2xx, иначе false
//	Error        - Ошибка если проверка не удалась
//
// Определение статусов:
//
//	UP           - 2xx статус код, нет ошибки
//	DOWN         - 5xx, timeout, или ошибка соединения
//	DEGRADED     - Медленный ответ (определяется внешне)
package http_checker
