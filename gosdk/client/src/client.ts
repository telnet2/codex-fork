/**
 * Tool Server Client implementation.
 */

import {
  Session,
  Execution,
  ToolSpec,
  CreateSessionRequest,
  CreateSessionResponse,
  InvokeToolRequest,
  InvokeToolResponse,
  OutputQueryResponse,
  APIError,
  ExecutionResult,
  StreamCallbacks,
  ToolServerClientOptions,
  SSEEventTypes,
  SSEStartedEvent,
  SSEOutputEvent,
  SSECompletedEvent,
  SSEErrorEvent,
} from './types';

/**
 * Error thrown when the API returns an error response.
 */
export class ToolServerError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly details?: Record<string, unknown>
  ) {
    super(message);
    this.name = 'ToolServerError';
  }

  static fromAPIError(apiError: APIError): ToolServerError {
    return new ToolServerError(
      apiError.error.code,
      apiError.error.message,
      apiError.error.details
    );
  }
}

/**
 * Default maximum output size for truncation (32KB).
 */
const DEFAULT_MAX_OUTPUT_SIZE = 32 * 1024;

/**
 * Default request timeout (30 seconds).
 */
const DEFAULT_TIMEOUT = 30000;

/**
 * Tool Server Client for interacting with the tool server.
 */
export class ToolServerClient {
  private baseUrl: string;
  private sessionId?: string;
  private timeout: number;
  private onSessionCreated?: (sessionId: string) => void;

  constructor(options: ToolServerClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/$/, ''); // Remove trailing slash
    this.sessionId = options.sessionId;
    this.timeout = options.timeout ?? DEFAULT_TIMEOUT;
    this.onSessionCreated = options.onSessionCreated;
  }

  /**
   * Get the current session ID.
   */
  getSessionId(): string | undefined {
    return this.sessionId;
  }

  /**
   * Set the session ID.
   */
  setSessionId(sessionId: string): void {
    this.sessionId = sessionId;
  }

  /**
   * Make a request to the API.
   */
  private async request<T>(
    method: string,
    path: string,
    body?: unknown
  ): Promise<T> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(`${this.baseUrl}${path}`, {
        method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      const data = await response.json();

      if (!response.ok) {
        if (data.error) {
          throw ToolServerError.fromAPIError(data as APIError);
        }
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      return data as T;
    } finally {
      clearTimeout(timeoutId);
    }
  }

  // ==================== Session Management ====================

  /**
   * Create a new session.
   */
  async createSession(options?: CreateSessionRequest): Promise<CreateSessionResponse> {
    const response = await this.request<CreateSessionResponse>(
      'POST',
      '/api/v1/sessions',
      options ?? {}
    );

    this.sessionId = response.session_id;
    this.onSessionCreated?.(response.session_id);

    return response;
  }

  /**
   * Get session details.
   */
  async getSession(sessionId?: string): Promise<Session> {
    const id = sessionId ?? this.sessionId;
    if (!id) {
      throw new Error('No session ID provided');
    }

    return this.request<Session>('GET', `/api/v1/sessions/${id}`);
  }

  /**
   * Delete a session.
   */
  async deleteSession(sessionId?: string): Promise<void> {
    const id = sessionId ?? this.sessionId;
    if (!id) {
      throw new Error('No session ID provided');
    }

    await this.request<void>('DELETE', `/api/v1/sessions/${id}`);

    if (id === this.sessionId) {
      this.sessionId = undefined;
    }
  }

  /**
   * Update the current working directory for a session.
   */
  async updateCwd(cwd: string, sessionId?: string): Promise<Session> {
    const id = sessionId ?? this.sessionId;
    if (!id) {
      throw new Error('No session ID provided');
    }

    return this.request<Session>('PUT', `/api/v1/sessions/${id}/cwd`, { cwd });
  }

  /**
   * Close the current session.
   */
  async closeSession(): Promise<void> {
    if (this.sessionId) {
      await this.deleteSession(this.sessionId);
    }
  }

  // ==================== Tool Operations ====================

  /**
   * List available tools.
   */
  async listTools(): Promise<ToolSpec[]> {
    const response = await this.request<{ tools: ToolSpec[] }>(
      'GET',
      '/api/v1/tools'
    );
    return response.tools;
  }

  /**
   * Invoke a tool synchronously.
   */
  async invoke(
    toolName: string,
    args: Record<string, unknown>,
    maxOutputSize: number = DEFAULT_MAX_OUTPUT_SIZE
  ): Promise<ExecutionResult> {
    const request: InvokeToolRequest = {
      session_id: this.sessionId,
      arguments: args,
    };

    const response = await this.request<InvokeToolResponse>(
      'POST',
      `/api/v1/tools/${toolName}/invoke`,
      request
    );

    // Update session ID if a new one was created
    if (!this.sessionId && response.session_id) {
      this.sessionId = response.session_id;
      this.onSessionCreated?.(response.session_id);
    }

    const execution = response.execution;
    if (!execution) {
      throw new Error('No execution data in response');
    }

    // Get output
    let stdout = '';
    let stderr = '';
    let stdoutTruncated = false;
    let stderrTruncated = false;

    if (execution.stdout_bytes > 0) {
      const stdoutResponse = await this.getOutput(
        response.execution_id,
        'stdout',
        0,
        maxOutputSize
      );
      stdout = stdoutResponse.data;
      stdoutTruncated = stdoutResponse.has_more;
    }

    if (execution.stderr_bytes > 0) {
      const stderrResponse = await this.getOutput(
        response.execution_id,
        'stderr',
        0,
        maxOutputSize
      );
      stderr = stderrResponse.data;
      stderrTruncated = stderrResponse.has_more;
    }

    return {
      executionId: response.execution_id,
      sessionId: response.session_id,
      status: execution.status,
      exitCode: execution.exit_code,
      stdout,
      stderr,
      error: execution.error,
      stdoutTruncated,
      stderrTruncated,
    };
  }

  /**
   * Invoke a tool with streaming output.
   */
  async invokeStreaming(
    toolName: string,
    args: Record<string, unknown>,
    callbacks: StreamCallbacks
  ): Promise<ExecutionResult> {
    const request: InvokeToolRequest = {
      session_id: this.sessionId,
      arguments: args,
    };

    const response = await fetch(
      `${this.baseUrl}/api/v1/tools/${toolName}/invoke/stream`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(request),
      }
    );

    if (!response.ok) {
      const errorData = await response.json();
      if (errorData.error) {
        throw ToolServerError.fromAPIError(errorData as APIError);
      }
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const reader = response.body?.getReader();
    if (!reader) {
      throw new Error('No response body');
    }

    const decoder = new TextDecoder();
    let buffer = '';
    let currentEvent = '';
    let stdout = '';
    let stderr = '';
    let executionId = '';
    let sessionId = this.sessionId ?? '';
    let exitCode: number | undefined;
    let error: string | undefined;
    let durationMs: number | undefined;
    let status: ExecutionResult['status'] = 'running';

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (line.startsWith('event: ')) {
            currentEvent = line.slice(7).trim();
          } else if (line.startsWith('data: ')) {
            const dataStr = line.slice(6);
            try {
              const data = JSON.parse(dataStr);

              switch (currentEvent) {
                case SSEEventTypes.STARTED: {
                  const event = data as SSEStartedEvent;
                  executionId = event.execution_id;
                  sessionId = event.session_id;
                  if (!this.sessionId) {
                    this.sessionId = sessionId;
                    this.onSessionCreated?.(sessionId);
                  }
                  callbacks.onStart?.(event);
                  break;
                }
                case SSEEventTypes.STDOUT: {
                  const event = data as SSEOutputEvent;
                  stdout += event.chunk;
                  callbacks.onStdout?.(event.chunk, event.offset);
                  break;
                }
                case SSEEventTypes.STDERR: {
                  const event = data as SSEOutputEvent;
                  stderr += event.chunk;
                  callbacks.onStderr?.(event.chunk, event.offset);
                  break;
                }
                case SSEEventTypes.COMPLETED: {
                  const event = data as SSECompletedEvent;
                  exitCode = event.exit_code;
                  durationMs = event.duration_ms;
                  status = 'completed';
                  break;
                }
                case SSEEventTypes.ERROR: {
                  const event = data as SSEErrorEvent;
                  error = event.error;
                  durationMs = event.duration_ms;
                  status = 'failed';
                  break;
                }
              }
            } catch {
              // Ignore JSON parse errors
            }
          }
        }
      }
    } finally {
      reader.releaseLock();
    }

    const result: ExecutionResult = {
      executionId,
      sessionId,
      status,
      exitCode,
      stdout,
      stderr,
      error,
      durationMs,
    };

    if (error) {
      callbacks.onError?.(new Error(error));
    } else {
      callbacks.onComplete?.(result);
    }

    return result;
  }

  // ==================== Execution Management ====================

  /**
   * Get execution details.
   */
  async getExecution(executionId: string, sessionId?: string): Promise<Execution> {
    const sid = sessionId ?? this.sessionId;
    if (!sid) {
      throw new Error('No session ID provided');
    }

    return this.request<Execution>(
      'GET',
      `/api/v1/executions/${executionId}?session_id=${sid}`
    );
  }

  /**
   * Get execution output with pagination.
   */
  async getOutput(
    executionId: string,
    stream: 'stdout' | 'stderr',
    offset: number = 0,
    limit: number = DEFAULT_MAX_OUTPUT_SIZE,
    sessionId?: string
  ): Promise<OutputQueryResponse> {
    const sid = sessionId ?? this.sessionId;
    if (!sid) {
      throw new Error('No session ID provided');
    }

    const params = new URLSearchParams({
      session_id: sid,
      stream,
      offset: offset.toString(),
      limit: limit.toString(),
    });

    return this.request<OutputQueryResponse>(
      'GET',
      `/api/v1/executions/${executionId}/output?${params}`
    );
  }

  /**
   * Cancel a running execution.
   */
  async cancelExecution(executionId: string): Promise<void> {
    await this.request<{ status: string; message: string }>(
      'POST',
      `/api/v1/executions/${executionId}/cancel`
    );
  }

  // ==================== Health Checks ====================

  /**
   * Check if the server is healthy.
   */
  async health(): Promise<boolean> {
    try {
      const response = await this.request<{ status: string }>('GET', '/health');
      return response.status === 'ok';
    } catch {
      return false;
    }
  }

  /**
   * Check if the server is ready.
   */
  async ready(): Promise<boolean> {
    try {
      const response = await this.request<{ status: string }>('GET', '/ready');
      return response.status === 'ready';
    } catch {
      return false;
    }
  }
}
