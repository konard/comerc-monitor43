// Package infrastructure предоставляет внешние адаптеры для alert-service.
//
// Основные компоненты:
//   - Субпакеты с конкретными адаптерами:
//   - auth: JWT аутентификация и gRPC interceptors
//   - postgres: PostgreSQL соединения и пулинг
//   - channels: Клиенты для внешних сервисов (Telegram, Email, Webhook)
//
// Использование:
//
//	authenticator := auth.NewAuthenticator(jwtSecret)
//	authMiddleware := auth.NewAuthMiddleware(jwtSecret)
//	grpcServer := grpc.NewServer(
//	    grpc.ChainUnaryInterceptor(
//	        authMiddleware.UnaryInterceptor(),
//	    ),
//	)
//
// Ключевые особенности:
//   - Аутентификация через JWT токены
//   - gRPC interceptors для request/response обработки
//   - Клиенты для внешних сервисов с retry логикой
//   - Context propagation для распределённых запросов
//
// Контекст DOD:
//   - Соответствует требованию DOD 1.5 Dependency Injection Patterns
//   - Использование pure-golang adapters для внешних зависимостей
//   - Интерфейсы для тестируемости и гибкости
//   - Proper error handling с контекстом
package infrastructure
