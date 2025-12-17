# Tool Server

A Go-based HTTP server that exposes gosdk tools via REST API with SSE streaming support.

## Features

- **Session Management**: Stateful sessions with persistent file storage
- **Tool Execution**: Execute gosdk tools with execution tracking
- **SSE Streaming**: Real-time output streaming for long-running commands
- **Large Output Handling**: Pagination support for large outputs
- **Automatic Cleanup**: Background job removes expired sessions

## Quick Start

### Build

```bash
cd gosdk/server
go build ./cmd/tool-server
```

### Run

```bash
./tool-server --port 8080
```

### Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `0.0.0.0` | Host to listen on |
| `--port` | `8080` | Port to listen on |
| `--temp-dir` | System temp | Base directory for session data |
| `--session-timeout` | `72` | Session timeout in hours |
| `--cleanup-interval` | `60` | Cleanup interval in minutes |

## API Endpoints

### Health

- `GET /health` - Health check
- `GET /ready` - Readiness check

### Sessions

- `POST /api/v1/sessions` - Create a new session
- `GET /api/v1/sessions/:id` - Get session details
- `DELETE /api/v1/sessions/:id` - Delete a session
- `PUT /api/v1/sessions/:id/cwd` - Update working directory

### Tools

- `GET /api/v1/tools` - List available tools
- `POST /api/v1/tools/:name/invoke` - Invoke a tool
- `POST /api/v1/tools/:name/invoke/stream` - Invoke with SSE streaming

### Executions

- `GET /api/v1/executions/:id?session_id=...` - Get execution details
- `GET /api/v1/executions/:id/output?session_id=...&stream=stdout` - Get output
- `POST /api/v1/executions/:id/cancel` - Cancel execution

## Example Usage

### Create Session

```bash
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"cwd": "/home/user/project"}'
```

### List Tools

```bash
curl http://localhost:8080/api/v1/tools
```

### Invoke Tool

```bash
curl -X POST http://localhost:8080/api/v1/tools/read_file/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "01HXYZ...",
    "arguments": {"file_path": "/etc/hosts"}
  }'
```

### Streaming Invocation

```bash
curl -N -X POST http://localhost:8080/api/v1/tools/read_file/invoke/stream \
  -H "Content-Type: application/json" \
  -d '{"arguments": {"file_path": "/etc/hosts"}}'
```

## SSE Event Types

| Event | Description |
|-------|-------------|
| `started` | Execution began |
| `stdout` | Stdout chunk |
| `stderr` | Stderr chunk |
| `completed` | Execution completed successfully |
| `error` | Execution failed |

## Directory Structure

```
/tmp/tool-server/
└── sessions/
    └── <session-id>/
        ├── session.json
        ├── session.lock
        ├── workspace/
        └── executions/
            └── <execution-id>/
                ├── meta.json
                ├── stdout
                └── stderr
```

## Testing

```bash
go test ./...
```

## License

MIT
