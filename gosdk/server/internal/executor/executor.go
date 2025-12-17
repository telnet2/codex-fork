// Package executor handles tool execution with output capture and tracking.
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/anthropics/codex-fork/gosdk/handlers"
	"github.com/anthropics/codex-fork/gosdk/server/internal/session"
	"github.com/anthropics/codex-fork/gosdk/server/pkg/types"
	"github.com/anthropics/codex-fork/gosdk/tools"
)

// Executor handles tool execution within sessions.
type Executor struct {
	storage  *session.Storage
	registry *tools.ToolRegistry

	// Track running executions for cancellation
	mu         sync.RWMutex
	running    map[string]context.CancelFunc
}

// NewExecutor creates a new executor with the default tool registry.
func NewExecutor(storage *session.Storage) *Executor {
	registry := tools.NewToolRegistry()

	// Register file tools with handlers that don't require session context
	registry.Register(tools.ReadFileTool, handlers.NewReadFileHandler())
	registry.Register(tools.ListDirTool, handlers.NewListDirHandler())

	// Note: GrepFilesTool requires a working directory from session context
	// It will be created dynamically when the tool is invoked
	registry.RegisterSpec(tools.GrepFilesTool)

	// Register other tools without handlers (will be implemented later)
	registry.RegisterSpec(tools.ShellCommandTool)
	registry.RegisterSpec(tools.PlanTool)
	registry.RegisterSpec(tools.ApplyPatchFreeformTool)

	return &Executor{
		storage:  storage,
		registry: registry,
		running:  make(map[string]context.CancelFunc),
	}
}

// GetRegistry returns the tool registry.
func (e *Executor) GetRegistry() *tools.ToolRegistry {
	return e.registry
}

// createDynamicHandler creates a handler for tools that require session context.
func (e *Executor) createDynamicHandler(toolName string, sess *types.Session) tools.ToolHandler {
	switch toolName {
	case "grep_files":
		// Use the session's working directory for grep
		return handlers.NewGrepFilesHandler(sess.Cwd)
	default:
		return nil
	}
}

// Execute runs a tool and returns the execution record.
func (e *Executor) Execute(ctx context.Context, sess *types.Session, req *types.InvokeToolRequest) (*types.Execution, error) {
	// Create execution record
	now := time.Now()
	exec := &types.Execution{
		ID:          types.NewULID(),
		SessionID:   sess.ID,
		ToolName:    req.ToolName,
		Arguments:   req.Arguments,
		Status:      types.ExecutionStatusPending,
		StartedAt:   now,
		StdoutBytes: 0,
		StderrBytes: 0,
	}

	// Create execution directory
	if err := e.storage.CreateExecutionDir(sess.ID, exec.ID); err != nil {
		return nil, fmt.Errorf("failed to create execution directory: %w", err)
	}

	// Save initial execution state
	if err := e.storage.SaveExecution(exec); err != nil {
		return nil, fmt.Errorf("failed to save execution: %w", err)
	}

	// Get the tool handler
	handler, ok := e.registry.GetHandler(req.ToolName)
	if !ok {
		// Check if this is a tool that requires dynamic handler creation
		handler = e.createDynamicHandler(req.ToolName, sess)
		if handler == nil {
			exec.Status = types.ExecutionStatusFailed
			exec.Error = fmt.Sprintf("unknown tool or no handler: %s", req.ToolName)
			e.storage.SaveExecution(exec)
			return exec, nil
		}
	}

	// Create cancellable context
	execCtx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.running[exec.ID] = cancel
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.running, exec.ID)
		e.mu.Unlock()
	}()

	// Update status to running
	exec.Status = types.ExecutionStatusRunning
	e.storage.SaveExecution(exec)

	// Execute the tool
	var output *tools.ToolOutput
	var execErr error

	done := make(chan struct{})
	go func() {
		defer close(done)
		output, execErr = handler.Handle(string(req.Arguments))
	}()

	select {
	case <-execCtx.Done():
		exec.Status = types.ExecutionStatusCancelled
		exec.Error = "execution cancelled"
	case <-done:
		if execErr != nil {
			exec.Status = types.ExecutionStatusFailed
			exec.Error = execErr.Error()
		} else if output != nil {
			// Write output to files
			if err := e.writeOutput(sess.ID, exec.ID, output); err != nil {
				exec.Status = types.ExecutionStatusFailed
				exec.Error = fmt.Sprintf("failed to write output: %v", err)
			} else {
				// Check success flag
				if output.Success != nil && !*output.Success {
					exec.Status = types.ExecutionStatusFailed
				} else {
					exec.Status = types.ExecutionStatusCompleted
					exitCode := 0
					exec.ExitCode = &exitCode
				}

				// Update byte counts
				exec.StdoutBytes = int64(len(output.Content))
			}
		}
	}

	// Update completion time
	completedAt := time.Now()
	exec.CompletedAt = &completedAt

	// Save final execution state
	if err := e.storage.SaveExecution(exec); err != nil {
		return nil, fmt.Errorf("failed to save execution: %w", err)
	}

	return exec, nil
}

// writeOutput writes tool output to files.
func (e *Executor) writeOutput(sessionID, executionID string, output *tools.ToolOutput) error {
	stdoutFile := e.storage.StdoutFile(sessionID, executionID)

	// Write stdout
	if err := os.WriteFile(stdoutFile, []byte(output.Content), 0644); err != nil {
		return fmt.Errorf("failed to write stdout: %w", err)
	}

	// If there are content items, append them as JSON
	if len(output.ContentItems) > 0 {
		itemsJSON, err := json.Marshal(output.ContentItems)
		if err == nil {
			f, err := os.OpenFile(stdoutFile, os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString("\n---\nContent Items:\n")
				f.Write(itemsJSON)
				f.Close()
			}
		}
	}

	return nil
}

// Cancel cancels a running execution.
func (e *Executor) Cancel(executionID string) bool {
	e.mu.RLock()
	cancel, ok := e.running[executionID]
	e.mu.RUnlock()

	if ok {
		cancel()
		return true
	}
	return false
}

// GetExecution retrieves an execution record.
func (e *Executor) GetExecution(sessionID, executionID string) (*types.Execution, error) {
	return e.storage.LoadExecution(sessionID, executionID)
}

// GetOutput retrieves execution output with pagination.
func (e *Executor) GetOutput(sessionID, executionID, stream string, offset, limit int64) (*types.OutputQueryResponse, error) {
	var filePath string
	switch stream {
	case "stdout":
		filePath = e.storage.StdoutFile(sessionID, executionID)
	case "stderr":
		filePath = e.storage.StderrFile(sessionID, executionID)
	default:
		return nil, fmt.Errorf("invalid stream: %s", stream)
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &types.OutputQueryResponse{
				Data:       "",
				Offset:     offset,
				TotalBytes: 0,
				HasMore:    false,
			}, nil
		}
		return nil, fmt.Errorf("failed to open output file: %w", err)
	}
	defer file.Close()

	// Get file size
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat output file: %w", err)
	}
	totalBytes := info.Size()

	// Seek to offset
	if offset > 0 {
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to seek: %w", err)
		}
	}

	// Set default limit
	if limit <= 0 {
		limit = 32 * 1024 // 32KB default
	}

	// Read data
	buf := make([]byte, limit)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read output: %w", err)
	}

	return &types.OutputQueryResponse{
		Data:       string(buf[:n]),
		Offset:     offset,
		TotalBytes: totalBytes,
		HasMore:    offset+int64(n) < totalBytes,
	}, nil
}
