package monitor_client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMonitorClient_ValidAddress(t *testing.T) {
	t.Parallel()

	// grpc.NewClient ленивый — не подключается сразу,
	// поэтому несуществующий адрес не вызывает ошибку при создании
	client, err := NewMonitorClient("localhost:9999")

	require.NoError(t, err)
	require.NotNil(t, client)

	// закрытие должно пройти без ошибок
	err = client.Close()
	assert.NoError(t, err)
}

func TestNewMonitorClient_EmptyAddress(t *testing.T) {
	t.Parallel()

	// пустой адрес тоже является допустимым для NewClient (lazy dial)
	client, err := NewMonitorClient("")

	// NewClient может вернуть ошибку или нет в зависимости от версии gRPC
	if err != nil {
		assert.Nil(t, client)
	} else {
		require.NotNil(t, client)
		assert.NoError(t, client.Close())
	}
}
