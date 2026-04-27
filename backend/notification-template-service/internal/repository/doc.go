// Package repository предоставляет репозитории для работы с хранилищем.
//
// Репозитории:
//
//	Postgres - реализация TemplateRepository для PostgreSQL
//
// Конвенции:
//   - Использует named queries через github.com/jmoiron/sqlx
//   - Всегда использует context для всех операций
//   - Возвращает domain модели из пакета model
//   - Ошибки оборачиваются с контекстом
package repository
