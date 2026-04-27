package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TelegramSender интерфейс для отправки сообщений в Telegram (используется для тестирования)
type TelegramSender interface {
	Send(ctx context.Context, chatID, message string) error
}

// TelegramClient отправляет уведомления в Telegram
type TelegramClient struct {
	botToken string
	apiURL   string
	client   *http.Client
}

// NewTelegramClient создаёт новый Telegram клиент
func NewTelegramClient(botToken, apiURL string) *TelegramClient {
	return &TelegramClient{
		botToken: botToken,
		apiURL:   apiURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send отправляет сообщение в Telegram
func (c *TelegramClient) Send(ctx context.Context, chatID, message string) (err error) {
	url := fmt.Sprintf("%s/bot%s/sendMessage", c.apiURL, c.botToken)
	payload := map[string]any{
		"chat_id": chatID,
		"text":    message,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer closeResponseBody(resp, &err, "failed to close telegram response body")
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}
	return nil
}
