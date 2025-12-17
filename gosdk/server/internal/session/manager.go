package session

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/anthropics/codex-fork/gosdk/server/pkg/types"
	"github.com/gofrs/flock"
)

const (
	// DefaultSessionTimeout is the default session expiration time (3 days).
	DefaultSessionTimeout = 72 * time.Hour

	// LockTimeout is the timeout for acquiring a session lock.
	LockTimeout = 5 * time.Second
)

// Manager handles session lifecycle operations.
type Manager struct {
	storage        *Storage
	sessionTimeout time.Duration

	// In-memory cache of active sessions for quick access
	mu       sync.RWMutex
	sessions map[string]*types.Session
}

// NewManager creates a new session manager.
func NewManager(baseDir string, sessionTimeout time.Duration) (*Manager, error) {
	if sessionTimeout <= 0 {
		sessionTimeout = DefaultSessionTimeout
	}

	storage := NewStorage(baseDir)
	if err := storage.EnsureBaseDir(); err != nil {
		return nil, fmt.Errorf("failed to ensure base directory: %w", err)
	}

	m := &Manager{
		storage:        storage,
		sessionTimeout: sessionTimeout,
		sessions:       make(map[string]*types.Session),
	}

	// Load existing sessions into cache
	if err := m.loadExistingSessions(); err != nil {
		return nil, fmt.Errorf("failed to load existing sessions: %w", err)
	}

	return m, nil
}

// Storage returns the underlying storage.
func (m *Manager) Storage() *Storage {
	return m.storage
}

// loadExistingSessions loads all existing sessions into the cache.
func (m *Manager) loadExistingSessions() error {
	ids, err := m.storage.ListSessionIDs()
	if err != nil {
		return err
	}

	for _, id := range ids {
		session, err := m.storage.LoadSession(id)
		if err != nil {
			// Log error but continue loading other sessions
			continue
		}
		if session != nil {
			m.sessions[id] = session
		}
	}
	return nil
}

// CreateSession creates a new session with the given parameters.
func (m *Manager) CreateSession(ctx context.Context, req *types.CreateSessionRequest) (*types.Session, error) {
	now := time.Now()
	sessionID := types.NewULID()

	// Create session directory structure
	if err := m.storage.CreateSessionDir(sessionID); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	workspaceDir := m.storage.WorkspaceDir(sessionID)

	// Determine working directory
	cwd := req.Cwd
	if cwd == "" {
		cwd = workspaceDir
	}

	session := &types.Session{
		ID:           sessionID,
		CreatedAt:    now,
		LastAccessAt: now,
		ExpiresAt:    now.Add(m.sessionTimeout),
		Cwd:          cwd,
		Env:          req.Env,
		WorkspaceDir: workspaceDir,
	}

	// Acquire lock and save session
	lock := m.storage.NewSessionLock(sessionID)
	if err := m.withLock(ctx, lock, func() error {
		return m.storage.SaveSession(session)
	}); err != nil {
		// Cleanup on failure
		_ = m.storage.DeleteSession(sessionID)
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Update cache
	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID.
func (m *Manager) GetSession(ctx context.Context, sessionID string) (*types.Session, error) {
	// Check cache first
	m.mu.RLock()
	session, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if exists {
		// Check if expired
		if time.Now().After(session.ExpiresAt) {
			return nil, nil
		}
		return session, nil
	}

	// Load from disk
	session, err := m.storage.LoadSession(sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		return nil, nil
	}

	// Update cache
	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	return session, nil
}

// TouchSession updates the session's last access time and extends expiry.
func (m *Manager) TouchSession(ctx context.Context, sessionID string) error {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	now := time.Now()
	session.LastAccessAt = now
	session.ExpiresAt = now.Add(m.sessionTimeout)

	lock := m.storage.NewSessionLock(sessionID)
	if err := m.withLock(ctx, lock, func() error {
		return m.storage.SaveSession(session)
	}); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	// Update cache
	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	return nil
}

// UpdateCwd updates the session's current working directory.
func (m *Manager) UpdateCwd(ctx context.Context, sessionID, cwd string) error {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Validate that cwd exists and is a directory
	info, err := os.Stat(cwd)
	if err != nil {
		return fmt.Errorf("invalid directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", cwd)
	}

	session.Cwd = cwd
	now := time.Now()
	session.LastAccessAt = now
	session.ExpiresAt = now.Add(m.sessionTimeout)

	lock := m.storage.NewSessionLock(sessionID)
	if err := m.withLock(ctx, lock, func() error {
		return m.storage.SaveSession(session)
	}); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	// Update cache
	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	return nil
}

// DeleteSession removes a session and all its data.
func (m *Manager) DeleteSession(ctx context.Context, sessionID string) error {
	// Remove from cache first
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()

	// Delete from storage
	if err := m.storage.DeleteSession(sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// ListSessions returns all non-expired sessions.
func (m *Manager) ListSessions(ctx context.Context) ([]*types.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []*types.Session
	now := time.Now()
	for _, session := range m.sessions {
		if now.Before(session.ExpiresAt) {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

// CleanupExpiredSessions removes all expired sessions.
func (m *Manager) CleanupExpiredSessions(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var expiredIDs []string

	for id, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			expiredIDs = append(expiredIDs, id)
		}
	}

	// Also check disk for sessions not in cache
	diskIDs, err := m.storage.ListSessionIDs()
	if err != nil {
		return 0, fmt.Errorf("failed to list sessions: %w", err)
	}

	for _, id := range diskIDs {
		if _, exists := m.sessions[id]; exists {
			continue // Already checked
		}
		session, err := m.storage.LoadSession(id)
		if err != nil {
			continue
		}
		if session != nil && now.After(session.ExpiresAt) {
			expiredIDs = append(expiredIDs, id)
		}
	}

	// Delete expired sessions
	deleted := 0
	for _, id := range expiredIDs {
		delete(m.sessions, id)
		if err := m.storage.DeleteSession(id); err != nil {
			continue
		}
		deleted++
	}

	return deleted, nil
}

// withLock executes a function while holding a file lock.
func (m *Manager) withLock(ctx context.Context, lock *flock.Flock, fn func() error) error {
	lockCtx, cancel := context.WithTimeout(ctx, LockTimeout)
	defer cancel()

	locked, err := lock.TryLockContext(lockCtx, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("could not acquire lock within timeout")
	}
	defer lock.Unlock()

	return fn()
}
