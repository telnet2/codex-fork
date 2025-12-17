package sse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventTypes(t *testing.T) {
	// Test that event type constants are defined
	assert.Equal(t, "started", EventStarted)
	assert.Equal(t, "stdout", EventStdout)
	assert.Equal(t, "stderr", EventStderr)
	assert.Equal(t, "completed", EventCompleted)
	assert.Equal(t, "error", EventError)
}

func TestStartedEvent(t *testing.T) {
	event := &StartedEvent{
		ExecutionID: "exec123",
		SessionID:   "sess456",
		ToolName:    "read_file",
		StartedAt:   time.Now(),
	}

	assert.Equal(t, "exec123", event.ExecutionID)
	assert.Equal(t, "sess456", event.SessionID)
	assert.Equal(t, "read_file", event.ToolName)
	assert.False(t, event.StartedAt.IsZero())
}

func TestOutputEvent(t *testing.T) {
	event := &OutputEvent{
		ExecutionID: "exec123",
		Chunk:       "Hello, World!",
		Offset:      0,
		Stream:      "stdout",
	}

	assert.Equal(t, "exec123", event.ExecutionID)
	assert.Equal(t, "Hello, World!", event.Chunk)
	assert.Equal(t, int64(0), event.Offset)
	assert.Equal(t, "stdout", event.Stream)
}

func TestCompletedEvent(t *testing.T) {
	event := &CompletedEvent{
		ExecutionID: "exec123",
		ExitCode:    0,
		DurationMs:  1000,
		StdoutBytes: 100,
		StderrBytes: 0,
	}

	assert.Equal(t, "exec123", event.ExecutionID)
	assert.Equal(t, 0, event.ExitCode)
	assert.Equal(t, int64(1000), event.DurationMs)
	assert.Equal(t, int64(100), event.StdoutBytes)
	assert.Equal(t, int64(0), event.StderrBytes)
}

func TestErrorEvent(t *testing.T) {
	event := &ErrorEvent{
		ExecutionID: "exec123",
		Error:       "file not found",
		DurationMs:  50,
	}

	assert.Equal(t, "exec123", event.ExecutionID)
	assert.Equal(t, "file not found", event.Error)
	assert.Equal(t, int64(50), event.DurationMs)
}
