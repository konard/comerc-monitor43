// Package grpc реализует gRPC handlers для мониторинговой системы.
//
// Пакет содержит gRPC-обработчики, которые:
//   - Извлекают данные из protobuf запросов
//   - Делегируют бизнес-логику сервисному слою
//   - Конвертируют результаты обратно в protobuf
//   - Выполняют авторизацию через middleware
//
// Основные handlers:
//   - MonitorHandler: Управление мониторами
//   - CheckResultHandler: Результаты проверок
//   - IncidentHandler: Инциденты
//   - MaintenanceHandler: Окна технического обслуживания
//
// Middleware:
//   - Auth middleware: JWT валидация и извлечение user_id/role/tier из контекста
//   - Logging middleware: Логирование всех gRPC запросов
//   - Tracing middleware: OpenTelemetry trace propagation
//   - Recovery middleware: Обработка panic
//
// Конвенции:
//   - Handlers НЕ содержат бизнес-логики
//   - Конвертация protobuf ↔ DTO происходит в handler
//   - Auth middleware должен быть вызван до handler
//   - Ошибки конвертируются в правильные gRPC статусы
//
// Пример использования:
//
//	handler := grpc.NewMonitorHandler(monitorService)
//	grpcServer := grpc.NewServer(
//	    grpc.ChainUnaryInterceptor(
//	        middleware.AuthInterceptor,
//	        middleware.LoggingInterceptor,
//	    ),
//	)
//	pb.RegisterMonitorServiceServer(grpcServer, handler)
package grpc
