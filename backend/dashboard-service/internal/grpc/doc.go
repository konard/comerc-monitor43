// Package grpc предоставляет gRPC обработчики dashboard-service.
//
// Компоненты:
//   - DashboardServiceServer: реализация DashboardService gRPC сервиса
//   - Конвертеры: преобразования protobuf ↔ domain модели
//   - Interceptors: middleware для трейсинга и метрик
//
// Использование:
//
//	server := grpc.NewDashboardServiceServer(dashboardSvc, historySvc, authMiddleware)
//	dashboardproto.RegisterDashboardServiceServer(gRPCServer, server)
package grpc
