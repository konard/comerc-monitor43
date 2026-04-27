package interfaces

import (
	"context"
)

// DistributedLocker определяет интерфейс для распределённых блокировок.
type DistributedLocker interface {
	// TryLock пытается получить блокировку.
	// Возвращает (locked=true, releaseFunc, error) или (locked=false, nil, nil).
	TryLock(ctx context.Context, key string) (bool, func() error, error)
}
