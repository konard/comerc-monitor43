package postgres

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"
)

// DistributedLocker реализует распределённую блокировку через PostgreSQL advisory locks.
//
// Использование:
//
//	locker := postgres.NewDistributedLocker(db)
//	acquired, unlock, err := locker.TryLock(ctx, "scheduling")
//	if err != nil { ... }
//	if !acquired { return } // другая инстанция уже выполняет
//	defer unlock()
//	// ... критическая секция ...
//
// Ограничения:
//   - Потокобезопасность: да
//   - Блокировка удерживается до вызова unlock() или закрытия соединения
//   - Использует pg_try_advisory_lock (неблокирующая)
type DistributedLocker struct {
	db *sql.DB
}

func NewDistributedLocker(db *sql.DB) *DistributedLocker {
	return &DistributedLocker{db: db}
}

// TryLock пытается получить advisory lock по ключу.
// Возвращает acquired=true и функцию unlock если блокировка получена.
// acquired=false если блокировка уже занята другой инстанцией.
func (l *DistributedLocker) TryLock(ctx context.Context, key string) (acquired bool, unlock func() error, err error) {
	lockID := keyToInt64(key)

	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed to begin lock transaction: %w", err)
	}

	var locked bool
	err = tx.QueryRowContext(ctx, "SELECT pg_try_advisory_xact_lock($1::bigint)", lockID).Scan(&locked)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return false, nil, stderrors.Join(
				fmt.Errorf("failed to acquire advisory lock: %w", err),
				fmt.Errorf("failed to rollback lock transaction: %w", rollbackErr),
			)
		}
		return false, nil, fmt.Errorf("failed to acquire advisory lock: %w", err)
	}

	if !locked {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return false, nil, fmt.Errorf("failed to rollback unlocked advisory transaction: %w", rollbackErr)
		}
		return false, nil, nil
	}

	return true, func() error {
		return tx.Commit()
	}, nil
}

func keyToInt64(key string) int64 {
	h := int64(0)
	for _, c := range key {
		h = 31*h + int64(c)
	}
	if h == 0 {
		h = 1
	}
	return h
}
