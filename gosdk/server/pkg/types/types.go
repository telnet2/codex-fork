// Package types provides shared types for the tool server.
package types

import (
	"crypto/rand"
	"encoding/json"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	entropy     = ulid.Monotonic(rand.Reader, 0)
	entropyLock sync.Mutex
)

// NewULID generates a new monotonically increasing ULID.
func NewULID() string {
	entropyLock.Lock()
	defer entropyLock.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

// Session represents a tool server session.
type Session struct {
	ID           string            `json:"id"`
	CreatedAt    time.Time         `json:"created_at"`
	LastAccessAt time.Time         `json:"last_access_at"`
	ExpiresAt    time.Time         `json:"expires_at"`
	Cwd          string            `json:"cwd"`
	Env          map[string]string `json:"env,omitempty"`
	WorkspaceDir string            `json:"workspace_dir"`
}

// ExecutionStatus represents the status of a tool execution.
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// Execution represents a tool execution record.
type Execution struct {
	ID          string          `json:"id"`
	SessionID   string          `json:"session_id"`
	ToolName    string          `json:"tool_name"`
	Arguments   json.RawMessage `json:"arguments"`
	Status      ExecutionStatus `json:"status"`
	ExitCode    *int            `json:"exit_code,omitempty"`
	Error       string          `json:"error,omitempty"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`

	// Output metadata (actual content stored in files)
	StdoutBytes int64 `json:"stdout_bytes"`
	StderrBytes int64 `json:"stderr_bytes"`
}

// CreateSessionRequest is the request to create a new session.
type CreateSessionRequest struct {
	Cwd string            `json:"cwd,omitempty"`
	Env map[string]string `json:"env,omitempty"`
}

// CreateSessionResponse is the response after creating a session.
type CreateSessionResponse struct {
	SessionID    string    `json:"session_id"`
	WorkspaceDir string    `json:"workspace_dir"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// UpdateCwdRequest is the request to update session's current working directory.
type UpdateCwdRequest struct {
	Cwd string `json:"cwd"`
}

// InvokeToolRequest is the request to invoke a tool.
type InvokeToolRequest struct {
	SessionID string          `json:"session_id,omitempty"`
	ToolName  string          `json:"tool_name"`
	Arguments json.RawMessage `json:"arguments"`
	Stream    bool            `json:"stream,omitempty"`
}

// InvokeToolResponse is the response after invoking a tool.
type InvokeToolResponse struct {
	SessionID   string     `json:"session_id"`
	ExecutionID string     `json:"execution_id"`
	Execution   *Execution `json:"execution,omitempty"`
}

// OutputQueryRequest is the request to query execution output.
type OutputQueryRequest struct {
	Stream string `json:"stream"` // "stdout" or "stderr"
	Offset int64  `json:"offset"`
	Limit  int64  `json:"limit"`
}

// OutputQueryResponse is the response containing execution output.
type OutputQueryResponse struct {
	Data       string `json:"data"`
	Offset     int64  `json:"offset"`
	TotalBytes int64  `json:"total_bytes"`
	HasMore    bool   `json:"has_more"`
}

// ErrorCode represents an API error code.
type ErrorCode string

const (
	ErrorCodeSessionNotFound   ErrorCode = "SESSION_NOT_FOUND"
	ErrorCodeSessionExpired    ErrorCode = "SESSION_EXPIRED"
	ErrorCodeExecutionNotFound ErrorCode = "EXECUTION_NOT_FOUND"
	ErrorCodeToolNotFound      ErrorCode = "TOOL_NOT_FOUND"
	ErrorCodeInvalidArguments  ErrorCode = "INVALID_ARGUMENTS"
	ErrorCodeExecutionFailed   ErrorCode = "EXECUTION_FAILED"
	ErrorCodeInternalError     ErrorCode = "INTERNAL_ERROR"
)

// APIError represents a structured API error.
type APIError struct {
	Code    ErrorCode              `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ErrorResponse wraps an APIError for JSON responses.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// NewAPIError creates a new API error.
func NewAPIError(code ErrorCode, message string) *ErrorResponse {
	return &ErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
		},
	}
}
