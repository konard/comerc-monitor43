// Package adapters связывает репозитории с интерфейсами сервисного слоя.
//
// Компоненты:
//   - DashboardAdapter: реализует DashboardService через MonitorStatusRepository
//   - HistoryAdapter: реализует HistoryService через CheckHistoryRepository и IncidentRepository
//
// Использование:
//
//	adapter := adapters.NewDashboardAdapter(statusRepo, exportDir)
//	server := grpc.NewDashboardServiceServer(adapter, historyAdapter, authMiddleware)
package adapters
