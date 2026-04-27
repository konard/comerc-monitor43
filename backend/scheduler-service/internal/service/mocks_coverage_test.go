package service

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// Mock для DistributedLocker в coverage тестах
type mockDistributedLocker struct {
	mock.Mock
}

func (m *mockDistributedLocker) TryLock(ctx context.Context, key string) (bool, func() error, error) {
	args := m.Called(ctx, key)
	unlockFunc := func() error { return nil }
	if args.Get(1) != nil {
		if fn, ok := args.Get(1).(func() error); ok {
			unlockFunc = fn
		}
	}
	return args.Bool(0), unlockFunc, args.Error(2)
}
