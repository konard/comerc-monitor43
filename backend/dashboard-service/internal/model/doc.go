// Package model содержит доменные модели dashboard-service.
//
// Модели:
//   - MonitorStatusView: денормализованный статус монитора для дашборда
//   - CheckHistoryEntry: запись истории проверок
//   - Incident: инцидент монитора
//   - PeriodMetrics: агрегированные метрики за период
//   - DashboardFilter: фильтры для запросов дашборда
//
// Ограничения:
//   - Потокобезопасность: нет (immutable после создания)
//   - Без внешних зависимостей (кроме uuid и time)
package model
