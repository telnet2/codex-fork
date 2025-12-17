package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewULID(t *testing.T) {
	// Generate multiple ULIDs
	ids := make([]string, 100)
	for i := 0; i < 100; i++ {
		ids[i] = NewULID()
	}

	// All should be unique
	seen := make(map[string]bool)
	for _, id := range ids {
		assert.False(t, seen[id], "ULID should be unique")
		seen[id] = true
	}

	// All should be 26 characters (ULID format)
	for _, id := range ids {
		assert.Len(t, id, 26, "ULID should be 26 characters")
	}

	// Should be monotonically increasing (lexicographically sortable)
	for i := 1; i < len(ids); i++ {
		assert.True(t, ids[i] >= ids[i-1], "ULIDs should be monotonically increasing")
	}
}

func TestNewAPIError(t *testing.T) {
	err := NewAPIError(ErrorCodeSessionNotFound, "Session not found")

	assert.Equal(t, ErrorCodeSessionNotFound, err.Error.Code)
	assert.Equal(t, "Session not found", err.Error.Message)
	assert.Nil(t, err.Error.Details)
}
