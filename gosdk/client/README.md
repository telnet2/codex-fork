# Tool Server Client

TypeScript client for interacting with the Tool Server.

## Installation

```bash
npm install @anthropic/tool-server-client
```

## Quick Start

```typescript
import { ToolServerClient } from '@anthropic/tool-server-client';

const client = new ToolServerClient({
  baseUrl: 'http://localhost:8080',
});

// Create a session
const session = await client.createSession({ cwd: '/home/user/project' });

// List available tools
const tools = await client.listTools();
console.log('Available tools:', tools.map(t => t.name));

// Invoke a tool
const result = await client.invoke('read_file', {
  file_path: '/etc/hosts'
});

console.log('Output:', result.stdout);
```

## Features

- **Session Management**: Create, get, update, and delete sessions
- **Tool Invocation**: Synchronous and streaming execution
- **Pagination**: Utilities for handling large outputs
- **TypeScript**: Full type definitions included

## API

### Client Options

```typescript
interface ToolServerClientOptions {
  baseUrl: string;           // Server URL
  sessionId?: string;        // Reuse existing session
  timeout?: number;          // Request timeout (default: 30000ms)
  onSessionCreated?: (id: string) => void;
}
```

### Session Management

```typescript
// Create session
const session = await client.createSession({ cwd: '/path/to/dir' });

// Get session
const session = await client.getSession();

// Update working directory
await client.updateCwd('/new/path');

// Close session
await client.closeSession();
```

### Tool Operations

```typescript
// List tools
const tools = await client.listTools();

// Invoke tool (synchronous)
const result = await client.invoke('read_file', {
  file_path: '/etc/hosts'
});

// Invoke tool (streaming)
await client.invokeStreaming('read_file', { file_path: '/etc/hosts' }, {
  onStart: (event) => console.log('Started:', event.execution_id),
  onStdout: (chunk) => process.stdout.write(chunk),
  onStderr: (chunk) => process.stderr.write(chunk),
  onComplete: (result) => console.log('Done:', result.status),
  onError: (error) => console.error('Error:', error),
});
```

### Execution Management

```typescript
// Get execution status
const execution = await client.getExecution('exec-id');

// Get output with pagination
const output = await client.getOutput('exec-id', 'stdout', 0, 1024);

// Cancel execution
await client.cancelExecution('exec-id');
```

### Pagination Utilities

```typescript
import { getAllOutput, OutputPaginator } from '@anthropic/tool-server-client';

// Get all output
const output = await getAllOutput(client, executionId, 'stdout', {
  chunkSize: 32 * 1024,
  maxBytes: 1024 * 1024,
  onChunk: (chunk, offset, total) => {
    console.log(`Progress: ${offset}/${total}`);
  }
});

// Manual pagination
const paginator = new OutputPaginator(client, executionId, 'stdout');
while (paginator.hasMore()) {
  const chunk = await paginator.next();
  if (chunk) {
    console.log(chunk.data);
  }
}
```

## Types

The client exports comprehensive TypeScript types:

```typescript
import type {
  Session,
  Execution,
  ExecutionStatus,
  ExecutionResult,
  ToolSpec,
  StreamCallbacks,
  OutputQueryResponse,
} from '@anthropic/tool-server-client';
```

## Error Handling

```typescript
import { ToolServerError, ErrorCodes } from '@anthropic/tool-server-client';

try {
  await client.getSession('invalid-id');
} catch (error) {
  if (error instanceof ToolServerError) {
    if (error.code === ErrorCodes.SESSION_NOT_FOUND) {
      console.log('Session not found');
    }
  }
}
```

## Building

```bash
npm install
npm run build
```

## Testing

```bash
npm test
```

## License

MIT
