package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// Client — обёртка над Redis-клиентом с базовыми операциями.
type Client struct {
	client *redis.Client
}

// NewClient создаёт Client; соединение не устанавливается, I/O не выполняется.
func NewClient(addr string, db int) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		DB:           db,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		DialTimeout:  5 * time.Second,
		PoolTimeout:  4 * time.Second,
	})
	return &Client{client: rdb}
}

// Get возвращает строковое значение ключа.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Set записывает значение ключа с указанным TTL.
func (c *Client) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Delete удаляет указанные ключи.
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Scan возвращает ключи по паттерну.
func (c *Client) Scan(ctx context.Context, match string, count int64) ([]string, error) {
	var keys []string
	var cursor uint64

	for {
		batch, next, err := c.client.Scan(ctx, cursor, match, count).Result()
		if err != nil {
			return nil, fmt.Errorf("scan keys: %w", err)
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

// Close закрывает соединение с Redis.
func (c *Client) Close() error {
	return c.client.Close()
}

// Ping проверяет доступность Redis.
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
