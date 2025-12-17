# Tool Server Design Document

## Overview

This document describes the design for a Go-based Tool Server that exposes the gosdk tools via HTTP/SSE. The server manages sessions using the file system and provides a TypeScript client for integration with LLM-based assistants.

## Goals

1. **Session-based execution**: Stateful sessions with persistent storage for reconnection
2. **Execution tracking**: Every tool invocation gets a unique execution ID for tracking
3. **Stream support**: Real-time output streaming via SSE for long-running commands
4. **Large output handling**: Pagination support for retrieving large outputs
5. **Resilience**: Sessions survive connection drops; clients can reconnect
6. **Cleanup**: Automatic session cleanup after 3-day timeout

## Architecture

```
┌─────────────────┐     HTTP/SSE      ┌─────────────────────────────────┐
│  TypeScript     │◄────────────────► │        Tool Server (Go)         │
│  Client         │                   │  ┌─────────────────────────────┐│
│                 │                   │  │     Session Manager         ││
│  - Stream exec  │                   │  │  - Create/Resume sessions   ││
│  - Pagination   │                   │  │  - Session cleanup (3 days) ││
│  - Reconnect    │                   │  └─────────────────────────────┘│
└─────────────────┘                   │  ┌─────────────────────────────┐│
                                      │  │     Tool Executor           ││
                                      │  │  - Execute gosdk tools      ││
                                      │  │  - Track execution IDs      ││
                                      │  │  - Stream stdout/stderr     ││
                                      │  └─────────────────────────────┘│
                                      │  ┌─────────────────────────────┐│
                                      │  │     File System Storage     ││
                                      │  │  - Session state            ││
                                      │  │  - Execution outputs        ││
                                      │  │  - Running process tracking ││
                                      │  └─────────────────────────────┘│
                                      └─────────────────────────────────┘
                                                      │
                                                      ▼
                                      ┌─────────────────────────────────┐
                                      │     Workspace (temp dir)        │
                                      │  /<temp>/tool-server/           │
                                      │  └── sessions/                  │
                                      │      └── <session-id>/          │
                                      │          ├── session.json       │
                                      │          ├── workspace/         │
                                      │          └── executions/        │
                                      │              └── <exec-id>/     │
                                      │                  ├── meta.json  │
                                      │                  ├── stdout     │
                                      │                  └── stderr     │
                                      └─────────────────────────────────┘
```

## Technology Stack

- **HTTP Framework**: [cloudwego/hertz](https://github.com/cloudwego/hertz) - High-performance Go HTTP framework
- **SSE**: [hertz SSE extension](https://www.cloudwego.io/docs/hertz/tutorials/basic-feature/sse/) for streaming
- **ULID**: `github.com/oklog/ulid/v2` for session/execution IDs (monotonically increasing)
- **File locking**: `github.com/gofrs/flock` for concurrent session access
- **JSON**: Standard library `encoding/json`

### Why ULID over UUID?

[ULID](https://github.com/ulid/spec) (Universally Unique Lexicographically Sortable Identifier) provides:

1. **Monotonically increasing**: IDs generated in sequence are naturally ordered by creation time
2. **Lexicographically sortable**: Directory listings and database queries return chronological order
3. **Timestamp embedded**: First 48 bits encode millisecond timestamp (extractable)
4. **UUID compatible**: 128-bit, can be stored in UUID columns
5. **URL safe**: Base32 encoding (26 characters, no special chars)

Example: `01ARZ3NDEKTSV4RRFFQ69G5FAV`
- Timestamp: `01ARZ3NDEK` (first 10 chars)
- Randomness: `TSV4RRFFQ69G5FAV` (last 16 chars)

## Data Models

### Session

```go
type Session struct {
    ID           string            `json:"id"`
    CreatedAt    time.Time         `json:"created_at"`
    LastAccessAt time.Time         `json:"last_access_at"`
    ExpiresAt    time.Time         `json:"expires_at"`        // CreatedAt + 3 days
    Cwd          string            `json:"cwd"`               // Current working directory
    Env          map[string]string `json:"env,omitempty"`     // Environment variables
    WorkspaceDir string            `json:"workspace_dir"`     // Root workspace path
}
```

### Execution

```go
type ExecutionStatus string

const (
    ExecutionStatusPending   ExecutionStatus = "pending"
    ExecutionStatusRunning   ExecutionStatus = "running"
    ExecutionStatusCompleted ExecutionStatus = "completed"
    ExecutionStatusFailed    ExecutionStatus = "failed"
    ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

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
```

### Tool Invocation Request/Response

```go
type InvokeToolRequest struct {
    SessionID string          `json:"session_id,omitempty"` // Optional: creates new if empty
    ToolName  string          `json:"tool_name"`
    Arguments json.RawMessage `json:"arguments"`
    Stream    bool            `json:"stream,omitempty"`     // Enable SSE streaming
}

type InvokeToolResponse struct {
    SessionID   string     `json:"session_id"`
    ExecutionID string     `json:"execution_id"`
    Execution   *Execution `json:"execution,omitempty"` // Included when not streaming
}
```

### Output Query

```go
type OutputQueryRequest struct {
    SessionID   string `json:"session_id"`
    ExecutionID string `json:"execution_id"`
    Stream      string `json:"stream"`  // "stdout" or "stderr"
    Offset      int64  `json:"offset"`  // Byte offset
    Limit       int64  `json:"limit"`   // Max bytes to return (default: 32KB)
}

type OutputQueryResponse struct {
    Data       string `json:"data"`        // Base64 encoded or UTF-8 string
    Offset     int64  `json:"offset"`      // Current offset
    TotalBytes int64  `json:"total_bytes"` // Total available bytes
    HasMore    bool   `json:"has_more"`    // More data available
}
```

## API Endpoints

### Session Management

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/sessions` | Create a new session |
| GET | `/api/v1/sessions/:id` | Get session details |
| DELETE | `/api/v1/sessions/:id` | Close and cleanup session |
| PUT | `/api/v1/sessions/:id/cwd` | Update current working directory |

### Tool Operations

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/tools` | List available tools with schemas |
| POST | `/api/v1/tools/:name/invoke` | Invoke a tool (returns execution ID) |
| GET | `/api/v1/tools/:name/invoke` (SSE) | Invoke with streaming response |

### Execution Management

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/executions/:id` | Get execution status and metadata |
| GET | `/api/v1/executions/:id/output` | Query stdout/stderr with pagination |
| GET | `/api/v1/executions/:id/stream` (SSE) | Stream execution output |
| POST | `/api/v1/executions/:id/cancel` | Cancel running execution |

### Health

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |

## SSE Event Types

When streaming execution output, the server sends the following SSE events:

```
event: started
data: {"execution_id": "01HX9Z3NDEKTSV4RRFFQ69G5FA", "started_at": "2024-05-15T10:30:00Z"}

event: stdout
data: {"chunk": "base64-encoded-data", "offset": 0}

event: stderr
data: {"chunk": "base64-encoded-data", "offset": 0}

event: completed
data: {"execution_id": "01HX9Z3NDEKTSV4RRFFQ69G5FA", "exit_code": 0, "duration_ms": 1234}

event: error
data: {"execution_id": "01HX9Z3NDEKTSV4RRFFQ69G5FA", "error": "command not found"}
```

> **Note**: Execution IDs are ULIDs (e.g., `01HX9Z3NDEKTSV4RRFFQ69G5FA`), which are monotonically
> increasing and sortable. The timestamp can be extracted from the first 10 characters.

## File System Layout

```
<temp_dir>/tool-server/
├── sessions/
│   └── <session-id>/
│       ├── session.json          # Session metadata
│       ├── session.lock          # File lock for concurrent access
│       ├── workspace/            # Session's working directory
│       └── executions/
│           └── <execution-id>/
│               ├── meta.json     # Execution metadata
│               ├── stdout        # Raw stdout output
│               ├── stderr        # Raw stderr output
│               └── input.json    # Original request (for replay)
└── cleanup.lock                  # Global cleanup lock
```

## Session Lifecycle

```
┌─────────────┐
│   Client    │
│  connects   │
└──────┬──────┘
       │
       ▼
┌──────────────┐  No session_id   ┌─────────────────┐
│ POST /invoke │─────────────────►│ Create Session  │
│              │                  │ - Generate ID   │
│              │                  │ - Create dirs   │
│              │                  │ - Set expiry    │
└──────┬───────┘                  └────────┬────────┘
       │                                   │
       │  Has session_id                   │
       ▼                                   ▼
┌──────────────┐                  ┌─────────────────┐
│ Load Session │◄─────────────────│ Return session  │
│ from disk    │                  │ ID to client    │
└──────┬───────┘                  └─────────────────┘
       │
       ▼
┌──────────────┐
│ Execute Tool │
│ - Create exec│
│ - Run command│
│ - Store output
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Return       │
│ execution_id │
└──────────────┘
```

## Session Cleanup

A background goroutine runs periodically to clean up expired sessions:

```go
func (sm *SessionManager) StartCleanup(ctx context.Context, interval time.Duration) {
    ticker := time.NewTicker(interval)
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            sm.cleanupExpiredSessions()
        }
    }
}

func (sm *SessionManager) cleanupExpiredSessions() {
    // 1. List all session directories
    // 2. For each session, check session.json expiry
    // 3. If expired (> 3 days since last access), delete directory
    // 4. Use file locking to prevent race conditions
}
```

## Tool Execution Flow

```go
import (
    "github.com/oklog/ulid/v2"
    "crypto/rand"
    "sync"
)

var (
    entropy     = ulid.Monotonic(rand.Reader, 0)
    entropyLock sync.Mutex
)

// NewULID generates a new monotonically increasing ULID
func NewULID() string {
    entropyLock.Lock()
    defer entropyLock.Unlock()
    return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

func (e *Executor) Execute(ctx context.Context, session *Session, req *InvokeToolRequest) (*Execution, error) {
    // 1. Create execution record with monotonic ULID
    exec := &Execution{
        ID:        NewULID(),
        SessionID: session.ID,
        ToolName:  req.ToolName,
        Arguments: req.Arguments,
        Status:    ExecutionStatusPending,
        StartedAt: time.Now(),
    }

    // 2. Save execution metadata
    if err := e.saveExecution(exec); err != nil {
        return nil, err
    }

    // 3. Get tool handler from registry
    handler := e.registry.GetHandler(req.ToolName)
    if handler == nil {
        exec.Status = ExecutionStatusFailed
        exec.Error = "unknown tool"
        return exec, nil
    }

    // 4. Execute tool (with output capture)
    exec.Status = ExecutionStatusRunning
    e.saveExecution(exec)

    output, err := handler.Handle(string(req.Arguments))

    // 5. Update execution status
    if err != nil {
        exec.Status = ExecutionStatusFailed
        exec.Error = err.Error()
    } else {
        exec.Status = ExecutionStatusCompleted
        exitCode := 0
        exec.ExitCode = &exitCode
    }

    now := time.Now()
    exec.CompletedAt = &now
    e.saveExecution(exec)

    return exec, nil
}
```

## TypeScript Client Design

### Dependencies

```json
{
  "dependencies": {
    "ulid": "^2.3.0"
  }
}
```

The client uses [ulid](https://www.npmjs.com/package/ulid) for client-side ID generation when needed,
matching the server's ULID format.

### Client Interface

```typescript
interface ToolServerClientOptions {
  baseUrl: string;
  sessionId?: string;
  timeout?: number;
  onSessionCreated?: (sessionId: string) => void;
}

interface ExecutionResult {
  executionId: string;
  status: 'completed' | 'failed' | 'cancelled';
  exitCode?: number;
  stdout: string;
  stderr: string;
  error?: string;
  durationMs: number;
}

interface StreamCallbacks {
  onStdout?: (chunk: string) => void;
  onStderr?: (chunk: string) => void;
  onComplete?: (result: ExecutionResult) => void;
  onError?: (error: Error) => void;
}

class ToolServerClient {
  constructor(options: ToolServerClientOptions);

  // Session management
  getSessionId(): string | undefined;
  async createSession(): Promise<string>;
  async closeSession(): Promise<void>;

  // Tool operations
  async listTools(): Promise<ToolSpec[]>;
  async invoke(toolName: string, args: Record<string, unknown>): Promise<ExecutionResult>;
  async invokeStreaming(toolName: string, args: Record<string, unknown>, callbacks: StreamCallbacks): Promise<void>;

  // Output retrieval (for large outputs)
  async getOutput(executionId: string, stream: 'stdout' | 'stderr', offset?: number, limit?: number): Promise<OutputChunk>;

  // Execution management
  async getExecution(executionId: string): Promise<Execution>;
  async cancelExecution(executionId: string): Promise<void>;
}
```

### Streaming Implementation

```typescript
async invokeStreaming(
  toolName: string,
  args: Record<string, unknown>,
  callbacks: StreamCallbacks
): Promise<void> {
  const url = new URL(`/api/v1/tools/${toolName}/invoke`, this.baseUrl);
  url.searchParams.set('stream', 'true');
  if (this.sessionId) {
    url.searchParams.set('session_id', this.sessionId);
  }

  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ arguments: args }),
  });

  const reader = response.body!.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() || '';

    for (const line of lines) {
      if (line.startsWith('event: ')) {
        const eventType = line.slice(7);
        // Handle event type
      } else if (line.startsWith('data: ')) {
        const data = JSON.parse(line.slice(6));
        switch (data.type) {
          case 'stdout':
            callbacks.onStdout?.(atob(data.chunk));
            break;
          case 'stderr':
            callbacks.onStderr?.(atob(data.chunk));
            break;
          case 'completed':
            callbacks.onComplete?.(data);
            break;
        }
      }
    }
  }
}
```

### Large Output Handling

When output exceeds a threshold (e.g., 32KB), the client truncates and provides pagination:

```typescript
const MAX_OUTPUT_SIZE = 32 * 1024; // 32KB

async invoke(toolName: string, args: Record<string, unknown>): Promise<ExecutionResult> {
  const result = await this._invokeInternal(toolName, args);

  // Check if output was truncated
  if (result.stdoutBytes > MAX_OUTPUT_SIZE) {
    result.stdout = result.stdout.slice(0, MAX_OUTPUT_SIZE);
    result.stdoutTruncated = true;
    result.stdoutTotalBytes = result.stdoutBytes;
  }

  return result;
}

// LLM can use this to paginate through large outputs
async getOutput(
  executionId: string,
  stream: 'stdout' | 'stderr',
  offset: number = 0,
  limit: number = MAX_OUTPUT_SIZE
): Promise<OutputChunk> {
  const response = await fetch(
    `${this.baseUrl}/api/v1/executions/${executionId}/output?` +
    `stream=${stream}&offset=${offset}&limit=${limit}`
  );
  return response.json();
}
```

## Implementation Plan

### Phase 1: Core Infrastructure ✅ COMPLETE
1. ✅ Set up Go module with hertz dependency
2. ✅ Implement session manager with file system storage
3. ✅ Implement basic session CRUD endpoints
4. ✅ Add file locking for concurrent access
5. ✅ Write unit and API tests

### Phase 2: Tool Execution ✅ COMPLETE
1. ✅ Integrate gosdk tool registry
2. ✅ Implement tool invocation endpoint
3. ✅ Add execution tracking with output capture
4. ✅ Implement output pagination
5. ✅ Write tool execution tests

### Phase 3: Streaming ✅ COMPLETE
1. ✅ Add SSE support using hertz SSE extension
2. ✅ Implement streaming tool invocation endpoint (POST /api/v1/tools/:name/invoke/stream)
3. ✅ Add real-time stdout/stderr streaming via SSE events
4. ✅ Add SSE event types (started, stdout, stderr, completed, error)
5. ✅ Write streaming endpoint tests

### Phase 4: Session Lifecycle
1. Implement session cleanup background job
2. Add session expiry handling
3. Implement execution cancellation

### Phase 5: TypeScript Client
1. Create npm package structure
2. Implement client class with session management
3. Add streaming support with EventSource
4. Add pagination utilities
5. Add reconnection logic

### Phase 6: Testing & Documentation
1. Add unit tests for session manager
2. Add integration tests for API endpoints
3. Add end-to-end tests with TypeScript client
4. Write API documentation

## Directory Structure

```
gosdk/
├── docs/
│   └── tool-server-design.md     # This document
├── server/                        # New: Tool server implementation
│   ├── cmd/
│   │   └── tool-server/
│   │       └── main.go           # Server entry point
│   ├── internal/
│   │   ├── api/
│   │   │   ├── router.go         # Hertz router setup
│   │   │   ├── session.go        # Session endpoints
│   │   │   ├── tools.go          # Tool endpoints
│   │   │   └── execution.go      # Execution endpoints
│   │   ├── session/
│   │   │   ├── manager.go        # Session manager
│   │   │   ├── storage.go        # File system storage
│   │   │   └── cleanup.go        # Session cleanup
│   │   ├── executor/
│   │   │   ├── executor.go       # Tool executor
│   │   │   └── output.go         # Output capture/pagination
│   │   └── sse/
│   │       └── stream.go         # SSE streaming utilities
│   ├── pkg/
│   │   └── types/
│   │       └── types.go          # Shared types
│   └── go.mod                    # Server module
└── client/                        # New: TypeScript client
    ├── src/
    │   ├── index.ts              # Client entry point
    │   ├── client.ts             # ToolServerClient class
    │   ├── types.ts              # TypeScript types
    │   ├── streaming.ts          # SSE streaming utilities
    │   └── pagination.ts         # Output pagination
    ├── package.json
    ├── tsconfig.json
    └── README.md
```

## Security Considerations

> **Note**: Authentication will be implemented in a future phase.

Current security measures:
1. **Session isolation**: Each session has its own workspace directory
2. **Path validation**: All file operations validate paths are within workspace
3. **Execution sandboxing**: Shell commands run with limited permissions (via gosdk)
4. **Input validation**: All API inputs are validated before processing

Future authentication will add:
- API key or JWT-based authentication
- Session ownership verification
- Rate limiting
- Audit logging

## Configuration

```go
type Config struct {
    // Server
    Host string `json:"host" default:"0.0.0.0"`
    Port int    `json:"port" default:"8080"`

    // Storage
    TempDir        string        `json:"temp_dir"`         // Base temp directory
    SessionTimeout time.Duration `json:"session_timeout"`  // Default: 72h (3 days)

    // Cleanup
    CleanupInterval time.Duration `json:"cleanup_interval"` // Default: 1h

    // Limits
    MaxOutputSize int64 `json:"max_output_size"` // Default: 100MB per execution
    MaxSessions   int   `json:"max_sessions"`    // Default: 1000
}
```

## Error Handling

All API errors follow a consistent format:

```json
{
  "error": {
    "code": "SESSION_NOT_FOUND",
    "message": "Session with ID 'abc123' not found",
    "details": {}
  }
}
```

Error codes:
- `SESSION_NOT_FOUND`: Session does not exist or expired
- `SESSION_EXPIRED`: Session has exceeded timeout
- `EXECUTION_NOT_FOUND`: Execution ID not found
- `TOOL_NOT_FOUND`: Unknown tool name
- `INVALID_ARGUMENTS`: Tool arguments validation failed
- `EXECUTION_FAILED`: Tool execution failed
- `INTERNAL_ERROR`: Unexpected server error

## Metrics (Future)

Planned metrics for observability:
- `tool_server_sessions_active`: Current active sessions
- `tool_server_executions_total`: Total executions by tool and status
- `tool_server_execution_duration_seconds`: Execution duration histogram
- `tool_server_output_bytes_total`: Total output bytes by stream type
