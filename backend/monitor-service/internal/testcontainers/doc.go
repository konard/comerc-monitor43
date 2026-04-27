// Package testcontainers предоставляет утилиты для integration тестов.
//
// Пакет содержит setup функции для создания testcontainers
// для integration тестов с реальными зависимостями (PostgreSQL, RabbitMQ).
//
// Использование:
//
// connStr, err := testcontainers.SetupPostgreSQLContainer(ctx)
// require.NoError(t, err)
// defer testcontainers.TeardownPostgreSQLContainer(ctx, connStr)
//
// db, err := postgres.NewDB(connStr)
package testcontainers
