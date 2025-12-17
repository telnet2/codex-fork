// Package session provides session management for the tool server.
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/codex-fork/gosdk/server/pkg/types"
	"github.com/gofrs/flock"
)

// Storage handles file system operations for sessions.
type Storage struct {
	baseDir string
}

// NewStorage creates a new storage instance.
func NewStorage(baseDir string) *Storage {
	return &Storage{baseDir: baseDir}
}

// BaseDir returns the base directory for session storage.
func (s *Storage) BaseDir() string {
	return s.baseDir
}

// SessionDir returns the directory path for a session.
func (s *Storage) SessionDir(sessionID string) string {
	return filepath.Join(s.baseDir, "sessions", sessionID)
}

// SessionFile returns the path to session.json for a session.
func (s *Storage) SessionFile(sessionID string) string {
	return filepath.Join(s.SessionDir(sessionID), "session.json")
}

// SessionLockFile returns the path to session.lock for a session.
func (s *Storage) SessionLockFile(sessionID string) string {
	return filepath.Join(s.SessionDir(sessionID), "session.lock")
}

// WorkspaceDir returns the workspace directory for a session.
func (s *Storage) WorkspaceDir(sessionID string) string {
	return filepath.Join(s.SessionDir(sessionID), "workspace")
}

// ExecutionsDir returns the executions directory for a session.
func (s *Storage) ExecutionsDir(sessionID string) string {
	return filepath.Join(s.SessionDir(sessionID), "executions")
}

// ExecutionDir returns the directory for a specific execution.
func (s *Storage) ExecutionDir(sessionID, executionID string) string {
	return filepath.Join(s.ExecutionsDir(sessionID), executionID)
}

// ExecutionFile returns the path to meta.json for an execution.
func (s *Storage) ExecutionFile(sessionID, executionID string) string {
	return filepath.Join(s.ExecutionDir(sessionID, executionID), "meta.json")
}

// StdoutFile returns the path to stdout file for an execution.
func (s *Storage) StdoutFile(sessionID, executionID string) string {
	return filepath.Join(s.ExecutionDir(sessionID, executionID), "stdout")
}

// StderrFile returns the path to stderr file for an execution.
func (s *Storage) StderrFile(sessionID, executionID string) string {
	return filepath.Join(s.ExecutionDir(sessionID, executionID), "stderr")
}

// EnsureBaseDir creates the base directory structure.
func (s *Storage) EnsureBaseDir() error {
	sessionsDir := filepath.Join(s.baseDir, "sessions")
	return os.MkdirAll(sessionsDir, 0755)
}

// CreateSessionDir creates the directory structure for a new session.
func (s *Storage) CreateSessionDir(sessionID string) error {
	dirs := []string{
		s.SessionDir(sessionID),
		s.WorkspaceDir(sessionID),
		s.ExecutionsDir(sessionID),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// CreateExecutionDir creates the directory structure for a new execution.
func (s *Storage) CreateExecutionDir(sessionID, executionID string) error {
	dir := s.ExecutionDir(sessionID, executionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create execution directory: %w", err)
	}
	return nil
}

// SaveSession saves session metadata to disk.
func (s *Storage) SaveSession(session *types.Session) error {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	file := s.SessionFile(session.ID)
	if err := os.WriteFile(file, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}
	return nil
}

// LoadSession loads session metadata from disk.
func (s *Storage) LoadSession(sessionID string) (*types.Session, error) {
	file := s.SessionFile(sessionID)
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}
	var session types.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}
	return &session, nil
}

// DeleteSession removes a session directory and all its contents.
func (s *Storage) DeleteSession(sessionID string) error {
	dir := s.SessionDir(sessionID)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to delete session directory: %w", err)
	}
	return nil
}

// SessionExists checks if a session directory exists.
func (s *Storage) SessionExists(sessionID string) bool {
	dir := s.SessionDir(sessionID)
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// ListSessionIDs returns all session IDs in storage.
func (s *Storage) ListSessionIDs() ([]string, error) {
	sessionsDir := filepath.Join(s.baseDir, "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list sessions directory: %w", err)
	}
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}
	return ids, nil
}

// SaveExecution saves execution metadata to disk.
func (s *Storage) SaveExecution(exec *types.Execution) error {
	data, err := json.MarshalIndent(exec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal execution: %w", err)
	}
	file := s.ExecutionFile(exec.SessionID, exec.ID)
	if err := os.WriteFile(file, data, 0644); err != nil {
		return fmt.Errorf("failed to write execution file: %w", err)
	}
	return nil
}

// LoadExecution loads execution metadata from disk.
func (s *Storage) LoadExecution(sessionID, executionID string) (*types.Execution, error) {
	file := s.ExecutionFile(sessionID, executionID)
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read execution file: %w", err)
	}
	var exec types.Execution
	if err := json.Unmarshal(data, &exec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal execution: %w", err)
	}
	return &exec, nil
}

// ListExecutionIDs returns all execution IDs for a session.
func (s *Storage) ListExecutionIDs(sessionID string) ([]string, error) {
	execDir := s.ExecutionsDir(sessionID)
	entries, err := os.ReadDir(execDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list executions directory: %w", err)
	}
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}
	return ids, nil
}

// NewSessionLock creates a file lock for a session.
func (s *Storage) NewSessionLock(sessionID string) *flock.Flock {
	return flock.New(s.SessionLockFile(sessionID))
}
