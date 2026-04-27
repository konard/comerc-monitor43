package postgres

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// nullableUUID
// ---------------------------------------------------------------------------

func TestNullableUUID_nil(t *testing.T) {
	t.Parallel()

	result := nullableUUID(uuid.Nil)

	assert.Nil(t, result)
}

func TestNullableUUID_valid(t *testing.T) {
	t.Parallel()

	id := uuid.New()

	result := nullableUUID(id)

	assert.Equal(t, id, result)
}

// ---------------------------------------------------------------------------
// isDuplicateKey
// ---------------------------------------------------------------------------

func TestIsDuplicateKey_nil(t *testing.T) {
	t.Parallel()

	assert.False(t, isDuplicateKey(nil))
}

func TestIsDuplicateKey_non_pq_error(t *testing.T) {
	t.Parallel()

	assert.False(t, isDuplicateKey(assert.AnError))
}

// ---------------------------------------------------------------------------
// metadataToJSON
// ---------------------------------------------------------------------------

func TestMetadataToJSON_nil(t *testing.T) {
	t.Parallel()

	result := metadataToJSON(nil)

	assert.Equal(t, "{}", result)
}

func TestMetadataToJSON_empty(t *testing.T) {
	t.Parallel()

	result := metadataToJSON(map[string]string{})

	assert.Equal(t, "{}", result)
}

func TestMetadataToJSON_with_data(t *testing.T) {
	t.Parallel()

	result := metadataToJSON(map[string]string{"key": "value"})

	assert.Contains(t, result, "key")
	assert.Contains(t, result, "value")
}

// ---------------------------------------------------------------------------
// jsonToMetadata
// ---------------------------------------------------------------------------

func TestJsonToMetadata_empty_string(t *testing.T) {
	t.Parallel()

	result := jsonToMetadata("")

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestJsonToMetadata_empty_object(t *testing.T) {
	t.Parallel()

	result := jsonToMetadata("{}")

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestJsonToMetadata_with_data(t *testing.T) {
	t.Parallel()

	result := jsonToMetadata(`{"version":"1.0","env":"prod"}`)

	assert.Equal(t, "1.0", result["version"])
	assert.Equal(t, "prod", result["env"])
}

func TestJsonToMetadata_invalid_json(t *testing.T) {
	t.Parallel()

	result := jsonToMetadata("not-json")

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// keyToInt64
// ---------------------------------------------------------------------------

func TestKeyToInt64_consistent(t *testing.T) {
	t.Parallel()

	h1 := keyToInt64("scheduling")
	h2 := keyToInt64("scheduling")

	assert.Equal(t, h1, h2)
}

func TestKeyToInt64_different_keys(t *testing.T) {
	t.Parallel()

	h1 := keyToInt64("scheduling")
	h2 := keyToInt64("cleanup")

	assert.NotEqual(t, h1, h2)
}

func TestKeyToInt64_non_zero(t *testing.T) {
	t.Parallel()

	// пустая строка должна вернуть 1 (не 0)
	h := keyToInt64("")

	assert.Equal(t, int64(1), h)
}

func TestKeyToInt64_nonempty_non_zero(t *testing.T) {
	t.Parallel()

	h := keyToInt64("any-key")

	assert.NotEqual(t, int64(0), h)
}

// ---------------------------------------------------------------------------
// New* конструкторы — проверяют инициализацию без реальной БД
// ---------------------------------------------------------------------------

func TestNewWorkerRepository(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	repo := NewWorkerRepository(db)

	require.NotNil(t, repo)
}

func TestNewScheduledCheckRepository(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	repo := NewScheduledCheckRepository(db)

	require.NotNil(t, repo)
}

func TestNewAuditRepository(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	repo := NewAuditRepository(db)

	require.NotNil(t, repo)
}

func TestNewDistributedLocker(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	locker := NewDistributedLocker(db)

	require.NotNil(t, locker)
}
