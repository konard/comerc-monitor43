// Package domain содержит domain entities (бизнес-сущности) мониторинговой системы.
//
// Пакет реализует бизнес-логику и валидацию доменных моделей без внешних зависимостей.
// Все сущности должны быть независимы от инфраструктуры (база данных, очереди, HTTP).
//
// Основные сущности:
//   - Monitor: Мониторинг проверок HTTP endpoints
//   - CheckResult: Результат проверки монитора
//   - MaintenanceWindow: Окно технического обслуживания
//   - Incident: Инцидент монитора
//
// Валидация:
//   - Все конструкторы возвращают error при невалидных данных
//   - Методы обновления данных (Update*, Activate, Cancel) выполняют валидацию
//   - Бизнес-правила инкапсулированы в методах сущностей
//
// Пример использования:
//
//	monitor, err := domain.NewMonitor(
//	    userID,
//	    "My Monitor",
//	    "https://example.com",
//	    60*time.Second,
//	    domain.MonitorConfig{...},
//	)
//	if err != nil {
//	    return err
//	}
//
//	if err := monitor.Activate(); err != nil {
//	    return err
//	}
package domain
