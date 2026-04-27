// Package realtime управляет WebSocket соединениями для real-time обновлений.
//
// Использование:
//
//	hub := realtime.NewHub()
//	go hub.Run()
//	client := realtime.NewClient(userID, conn, hub)
//	hub.Register(client)
//
// Компоненты:
//   - Hub: управление подписками и рассылка сообщений
//   - Client: обёртка над WebSocket соединением
//
// Ограничения:
//   - Требует вызова Close() для каждого Client
//   - Hub.Run() должен запускаться в отдельной горутине
package realtime
