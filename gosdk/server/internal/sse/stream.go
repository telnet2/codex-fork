// Package sse provides Server-Sent Events utilities for streaming tool output.
package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/sse"
)

// Event types for SSE streaming
const (
	EventStarted   = "started"
	EventStdout    = "stdout"
	EventStderr    = "stderr"
	EventCompleted = "completed"
	EventError     = "error"
)

// StartedEvent is sent when execution begins.
type StartedEvent struct {
	ExecutionID string    `json:"execution_id"`
	SessionID   string    `json:"session_id"`
	ToolName    string    `json:"tool_name"`
	StartedAt   time.Time `json:"started_at"`
}

// OutputEvent is sent for stdout/stderr chunks.
type OutputEvent struct {
	ExecutionID string `json:"execution_id"`
	Chunk       string `json:"chunk"` // Base64 encoded for binary safety
	Offset      int64  `json:"offset"`
	Stream      string `json:"stream"` // "stdout" or "stderr"
}

// CompletedEvent is sent when execution finishes successfully.
type CompletedEvent struct {
	ExecutionID string  `json:"execution_id"`
	ExitCode    int     `json:"exit_code"`
	DurationMs  int64   `json:"duration_ms"`
	StdoutBytes int64   `json:"stdout_bytes"`
	StderrBytes int64   `json:"stderr_bytes"`
}

// ErrorEvent is sent when execution fails.
type ErrorEvent struct {
	ExecutionID string `json:"execution_id"`
	Error       string `json:"error"`
	DurationMs  int64  `json:"duration_ms"`
}

// Streamer handles SSE streaming to a client.
type Streamer struct {
	ctx    context.Context
	c      *app.RequestContext
	stream *sse.Stream
}

// NewStreamer creates a new SSE streamer.
func NewStreamer(ctx context.Context, c *app.RequestContext) *Streamer {
	return &Streamer{
		ctx:    ctx,
		c:      c,
		stream: sse.NewStream(c),
	}
}

// SendEvent sends an SSE event with the given type and data.
func (s *Streamer) SendEvent(eventType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	event := &sse.Event{
		Event: eventType,
		Data:  jsonData,
	}

	return s.stream.Publish(event)
}

// SendStarted sends a started event.
func (s *Streamer) SendStarted(event *StartedEvent) error {
	return s.SendEvent(EventStarted, event)
}

// SendStdout sends a stdout chunk event.
func (s *Streamer) SendStdout(executionID string, chunk []byte, offset int64) error {
	return s.SendEvent(EventStdout, &OutputEvent{
		ExecutionID: executionID,
		Chunk:       string(chunk),
		Offset:      offset,
		Stream:      "stdout",
	})
}

// SendStderr sends a stderr chunk event.
func (s *Streamer) SendStderr(executionID string, chunk []byte, offset int64) error {
	return s.SendEvent(EventStderr, &OutputEvent{
		ExecutionID: executionID,
		Chunk:       string(chunk),
		Offset:      offset,
		Stream:      "stderr",
	})
}

// SendCompleted sends a completed event.
func (s *Streamer) SendCompleted(event *CompletedEvent) error {
	return s.SendEvent(EventCompleted, event)
}

// SendError sends an error event.
func (s *Streamer) SendError(event *ErrorEvent) error {
	return s.SendEvent(EventError, event)
}

// Done checks if the streaming context is done.
func (s *Streamer) Done() <-chan struct{} {
	return s.ctx.Done()
}
