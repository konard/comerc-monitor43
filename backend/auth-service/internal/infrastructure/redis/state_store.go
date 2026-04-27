package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const oauthStateTTL = 15 * time.Minute

// StateStore реализует OAuth CSRF state store через Redis.
type StateStore struct {
	client *Client
}

// NewStateStore создаёт StateStore.
func NewStateStore(client *Client) *StateStore {
	return &StateStore{client: client}
}

func (s *StateStore) Set(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return s.client.Set(ctx, key, string(data), oauthStateTTL)
}

func (s *StateStore) Get(ctx context.Context, key string) (any, error) {
	val, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}
	return result, nil
}

func (s *StateStore) Delete(ctx context.Context, key string) error {
	return s.client.Delete(ctx, key)
}
