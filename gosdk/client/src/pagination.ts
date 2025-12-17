/**
 * Pagination utilities for handling large outputs.
 */

import { ToolServerClient } from './client';
import { OutputQueryResponse } from './types';

/**
 * Default chunk size for pagination (32KB).
 */
const DEFAULT_CHUNK_SIZE = 32 * 1024;

/**
 * Options for paginated output retrieval.
 */
export interface PaginationOptions {
  /** Chunk size in bytes (default: 32KB) */
  chunkSize?: number;
  /** Maximum total bytes to retrieve (default: unlimited) */
  maxBytes?: number;
  /** Callback for each chunk */
  onChunk?: (chunk: string, offset: number, totalBytes: number) => void;
}

/**
 * Result of paginated output retrieval.
 */
export interface PaginatedOutputResult {
  /** Complete output data */
  data: string;
  /** Total bytes retrieved */
  bytesRetrieved: number;
  /** Total bytes available */
  totalBytes: number;
  /** Whether output was truncated due to maxBytes limit */
  truncated: boolean;
}

/**
 * Iterator for paginated output.
 */
export class OutputPaginator {
  private client: ToolServerClient;
  private executionId: string;
  private stream: 'stdout' | 'stderr';
  private sessionId?: string;
  private chunkSize: number;
  private currentOffset: number = 0;
  private totalBytes?: number;
  private done: boolean = false;

  constructor(
    client: ToolServerClient,
    executionId: string,
    stream: 'stdout' | 'stderr',
    options?: {
      sessionId?: string;
      chunkSize?: number;
    }
  ) {
    this.client = client;
    this.executionId = executionId;
    this.stream = stream;
    this.sessionId = options?.sessionId;
    this.chunkSize = options?.chunkSize ?? DEFAULT_CHUNK_SIZE;
  }

  /**
   * Check if there's more data to fetch.
   */
  hasMore(): boolean {
    return !this.done;
  }

  /**
   * Get the current offset.
   */
  getOffset(): number {
    return this.currentOffset;
  }

  /**
   * Get the total bytes if known.
   */
  getTotalBytes(): number | undefined {
    return this.totalBytes;
  }

  /**
   * Fetch the next chunk of data.
   */
  async next(): Promise<OutputQueryResponse | null> {
    if (this.done) {
      return null;
    }

    const response = await this.client.getOutput(
      this.executionId,
      this.stream,
      this.currentOffset,
      this.chunkSize,
      this.sessionId
    );

    this.totalBytes = response.total_bytes;
    this.currentOffset = response.offset + response.data.length;
    this.done = !response.has_more;

    return response;
  }

  /**
   * Reset the paginator to the beginning.
   */
  reset(): void {
    this.currentOffset = 0;
    this.done = false;
  }

  /**
   * Seek to a specific offset.
   */
  seek(offset: number): void {
    this.currentOffset = offset;
    this.done = false;
  }
}

/**
 * Retrieve all output with pagination.
 *
 * @param client - Tool server client
 * @param executionId - Execution ID
 * @param stream - Stream type ('stdout' or 'stderr')
 * @param options - Pagination options
 * @returns Complete output result
 */
export async function getAllOutput(
  client: ToolServerClient,
  executionId: string,
  stream: 'stdout' | 'stderr',
  options?: PaginationOptions & { sessionId?: string }
): Promise<PaginatedOutputResult> {
  const chunkSize = options?.chunkSize ?? DEFAULT_CHUNK_SIZE;
  const maxBytes = options?.maxBytes;
  const onChunk = options?.onChunk;

  const chunks: string[] = [];
  let offset = 0;
  let totalBytes = 0;
  let hasMore = true;
  let bytesRetrieved = 0;

  while (hasMore) {
    // Check if we've reached the max bytes limit
    if (maxBytes !== undefined && bytesRetrieved >= maxBytes) {
      return {
        data: chunks.join(''),
        bytesRetrieved,
        totalBytes,
        truncated: true,
      };
    }

    // Calculate how many bytes to request
    const bytesToRequest = maxBytes !== undefined
      ? Math.min(chunkSize, maxBytes - bytesRetrieved)
      : chunkSize;

    const response = await client.getOutput(
      executionId,
      stream,
      offset,
      bytesToRequest,
      options?.sessionId
    );

    chunks.push(response.data);
    bytesRetrieved += response.data.length;
    totalBytes = response.total_bytes;
    offset = response.offset + response.data.length;
    hasMore = response.has_more;

    onChunk?.(response.data, response.offset, response.total_bytes);
  }

  return {
    data: chunks.join(''),
    bytesRetrieved,
    totalBytes,
    truncated: false,
  };
}

/**
 * Create an async iterator for output chunks.
 *
 * @param client - Tool server client
 * @param executionId - Execution ID
 * @param stream - Stream type ('stdout' or 'stderr')
 * @param options - Options
 * @returns Async generator of output responses
 */
export async function* iterateOutput(
  client: ToolServerClient,
  executionId: string,
  stream: 'stdout' | 'stderr',
  options?: {
    sessionId?: string;
    chunkSize?: number;
  }
): AsyncGenerator<OutputQueryResponse, void, unknown> {
  const paginator = new OutputPaginator(client, executionId, stream, options);

  while (paginator.hasMore()) {
    const chunk = await paginator.next();
    if (chunk) {
      yield chunk;
    }
  }
}
