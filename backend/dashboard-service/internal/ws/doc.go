// Package ws предоставляет HTTP→WebSocket upgrade handler.
//
// Использование:
//
//	handler := ws.NewWSHandler(hub, authenticator, logger, writeTimeout, pongTimeout, pingInterval, maxMsgSize)
//	http.Handle("/ws", handler)
//
// Аутентификация:
//   - JWT токен передаётся через query параметр ?token=xxx
//   - При невалидном токене соединение отклоняется (401)
package ws
