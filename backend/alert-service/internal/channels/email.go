package channels

import (
	"context"
	"fmt"
	"net/smtp"
)

// EmailSender интерфейс для отправки email (используется для тестирования)
type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// EmailClient отправляет email уведомления
type EmailClient struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewEmailClient создаёт новый Email клиент
func NewEmailClient(host string, port int, from string) *EmailClient {
	return &EmailClient{
		host: host,
		port: port,
		from: from,
	}
}

// Send отправляет email
func (c *EmailClient) Send(ctx context.Context, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		c.from, to, subject, body)

	errChan := make(chan error, 1)
	go func() {
		errChan <- smtp.SendMail(
			addr,
			nil,
			c.from,
			[]string{to},
			[]byte(msg),
		)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}
