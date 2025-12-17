/**
 * Tool Server Client
 *
 * TypeScript client for interacting with the Tool Server.
 *
 * @packageDocumentation
 */

// Main client
export { ToolServerClient, ToolServerError } from './client';

// Types
export type {
  Session,
  Execution,
  ExecutionStatus,
  ToolSpec,
  JSONSchema,
  CreateSessionRequest,
  CreateSessionResponse,
  InvokeToolRequest,
  InvokeToolResponse,
  OutputQueryResponse,
  APIError,
  ExecutionResult,
  StreamCallbacks,
  ToolServerClientOptions,
  SSEStartedEvent,
  SSEOutputEvent,
  SSECompletedEvent,
  SSEErrorEvent,
  SSEEventType,
  ErrorCode,
} from './types';

export { ErrorCodes, SSEEventTypes } from './types';

// Pagination utilities
export {
  OutputPaginator,
  getAllOutput,
  iterateOutput,
} from './pagination';

export type {
  PaginationOptions,
  PaginatedOutputResult,
} from './pagination';
