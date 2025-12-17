/**
 * Type definitions for the Tool Server client.
 */

/**
 * Session represents a tool server session.
 */
export interface Session {
  id: string;
  created_at: string;
  last_access_at: string;
  expires_at: string;
  cwd: string;
  env?: Record<string, string>;
  workspace_dir: string;
}

/**
 * Execution status enum.
 */
export type ExecutionStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

/**
 * Execution represents a tool execution.
 */
export interface Execution {
  id: string;
  session_id: string;
  tool_name: string;
  arguments: unknown;
  status: ExecutionStatus;
  exit_code?: number;
  error?: string;
  started_at: string;
  completed_at?: string;
  stdout_bytes: number;
  stderr_bytes: number;
}

/**
 * Tool specification.
 */
export interface ToolSpec {
  name: string;
  description: string;
  parameters?: JSONSchema;
}

/**
 * JSON Schema type (simplified).
 */
export interface JSONSchema {
  type?: string;
  properties?: Record<string, JSONSchema>;
  required?: string[];
  items?: JSONSchema;
  description?: string;
  [key: string]: unknown;
}

/**
 * Create session request.
 */
export interface CreateSessionRequest {
  cwd?: string;
  env?: Record<string, string>;
}

/**
 * Create session response.
 */
export interface CreateSessionResponse {
  session_id: string;
  workspace_dir: string;
  created_at: string;
  expires_at: string;
}

/**
 * Invoke tool request.
 */
export interface InvokeToolRequest {
  session_id?: string;
  arguments: unknown;
}

/**
 * Invoke tool response.
 */
export interface InvokeToolResponse {
  session_id: string;
  execution_id: string;
  execution?: Execution;
}

/**
 * Output query response.
 */
export interface OutputQueryResponse {
  data: string;
  offset: number;
  total_bytes: number;
  has_more: boolean;
}

/**
 * API error response.
 */
export interface APIError {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
  };
}

/**
 * Error codes from the server.
 */
export const ErrorCodes = {
  SESSION_NOT_FOUND: 'SESSION_NOT_FOUND',
  SESSION_EXPIRED: 'SESSION_EXPIRED',
  EXECUTION_NOT_FOUND: 'EXECUTION_NOT_FOUND',
  TOOL_NOT_FOUND: 'TOOL_NOT_FOUND',
  INVALID_ARGUMENTS: 'INVALID_ARGUMENTS',
  EXECUTION_FAILED: 'EXECUTION_FAILED',
  INTERNAL_ERROR: 'INTERNAL_ERROR',
} as const;

export type ErrorCode = (typeof ErrorCodes)[keyof typeof ErrorCodes];

/**
 * SSE event types.
 */
export const SSEEventTypes = {
  STARTED: 'started',
  STDOUT: 'stdout',
  STDERR: 'stderr',
  COMPLETED: 'completed',
  ERROR: 'error',
} as const;

export type SSEEventType = (typeof SSEEventTypes)[keyof typeof SSEEventTypes];

/**
 * SSE started event data.
 */
export interface SSEStartedEvent {
  execution_id: string;
  session_id: string;
  tool_name: string;
  started_at: string;
}

/**
 * SSE output event data (stdout/stderr).
 */
export interface SSEOutputEvent {
  execution_id: string;
  chunk: string;
  offset: number;
  stream: 'stdout' | 'stderr';
}

/**
 * SSE completed event data.
 */
export interface SSECompletedEvent {
  execution_id: string;
  exit_code: number;
  duration_ms: number;
  stdout_bytes: number;
  stderr_bytes: number;
}

/**
 * SSE error event data.
 */
export interface SSEErrorEvent {
  execution_id: string;
  error: string;
  duration_ms: number;
}

/**
 * Execution result returned from invoke operations.
 */
export interface ExecutionResult {
  executionId: string;
  sessionId: string;
  status: ExecutionStatus;
  exitCode?: number;
  stdout: string;
  stderr: string;
  error?: string;
  durationMs?: number;
  stdoutTruncated?: boolean;
  stderrTruncated?: boolean;
}

/**
 * Stream callbacks for streaming execution.
 */
export interface StreamCallbacks {
  onStart?: (event: SSEStartedEvent) => void;
  onStdout?: (chunk: string, offset: number) => void;
  onStderr?: (chunk: string, offset: number) => void;
  onComplete?: (result: ExecutionResult) => void;
  onError?: (error: Error) => void;
}

/**
 * Client options.
 */
export interface ToolServerClientOptions {
  /** Base URL of the tool server */
  baseUrl: string;
  /** Optional session ID to reuse */
  sessionId?: string;
  /** Request timeout in milliseconds (default: 30000) */
  timeout?: number;
  /** Callback when a new session is created */
  onSessionCreated?: (sessionId: string) => void;
}
