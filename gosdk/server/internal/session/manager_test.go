package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anthropics/codex-fork/gosdk/server/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	tempDir := t.TempDir()

	m, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)
	require.NotNil(t, m)

	// Verify base directory was created
	sessionsDir := filepath.Join(tempDir, "sessions")
	info, err := os.Stat(sessionsDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCreateSession(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	m, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	// Create session with default values
	session, err := m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)
	require.NotNil(t, session)

	// Verify session ID is a valid ULID (26 characters)
	assert.Len(t, session.ID, 26)

	// Verify timestamps
	assert.False(t, session.CreatedAt.IsZero())
	assert.False(t, session.LastAccessAt.IsZero())
	assert.False(t, session.ExpiresAt.IsZero())
	assert.True(t, session.ExpiresAt.After(session.CreatedAt))

	// Verify workspace directory exists
	info, err := os.Stat(session.WorkspaceDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify default cwd is workspace dir
	assert.Equal(t, session.WorkspaceDir, session.Cwd)
}

func TestCreateSessionWithCustomCwd(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	m, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	customCwd := t.TempDir()
	session, err := m.CreateSession(ctx, &types.CreateSessionRequest{
		Cwd: customCwd,
	})
	require.NoError(t, err)
	require.NotNil(t, session)

	assert.Equal(t, customCwd, session.Cwd)
}

func TestCreateSessionWithEnv(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	m, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	env := map[string]string{
		"FOO": "bar",
		"BAZ": "qux",
	}
	session, err := m.CreateSession(ctx, &types.CreateSessionRequest{
		Env: env,
	})
	require.NoError(t, err)
	require.NotNil(t, session)

	assert.Equal(t, env, session.Env)
}

func TestGetSession(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	m, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	// Create a session
	created, err := m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)

	// Retrieve it
	retrieved, err := m.GetSession(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.WorkspaceDir, retrieved.WorkspaceDir)
}

func TestGetNonExistentSession(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	m, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	session, err := m.GetSession(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, session)
}

func TestGetExpiredSession(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	// Create manager with very short timeout
	m, err := NewManager(tempDir, time.Millisecond)
	require.NoError(t, err)

	// Create a session
	created, err := m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)

	// Wait for it to expire
	time.Sleep(10 * time.Millisecond)

	// Try to retrieve it - should return nil (expired)
	retrieved, err := m.GetSession(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestDeleteSession(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	m, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	// Create a session
	session, err := m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)

	// Delete it
	err = m.DeleteSession(ctx, session.ID)
	require.NoError(t, err)

	// Verify it's gone
	retrieved, err := m.GetSession(ctx, session.ID)
	require.NoError(t, err)
	assert.Nil(t, retrieved)

	// Verify directory is removed
	_, err = os.Stat(m.storage.SessionDir(session.ID))
	assert.True(t, os.IsNotExist(err))
}

func TestUpdateCwd(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	m, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	// Create a session
	session, err := m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)

	// Update cwd to a valid directory
	newCwd := t.TempDir()
	err = m.UpdateCwd(ctx, session.ID, newCwd)
	require.NoError(t, err)

	// Verify the update
	updated, err := m.GetSession(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, newCwd, updated.Cwd)
}

func TestUpdateCwdInvalidPath(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	m, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	// Create a session
	session, err := m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)

	// Try to update cwd to a non-existent directory
	err = m.UpdateCwd(ctx, session.ID, "/nonexistent/path")
	assert.Error(t, err)
}

func TestListSessions(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	m, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	// Create multiple sessions
	_, err = m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)
	_, err = m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)
	_, err = m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)

	// List sessions
	sessions, err := m.ListSessions(ctx)
	require.NoError(t, err)
	assert.Len(t, sessions, 3)
}

func TestCleanupExpiredSessions(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	// Create manager with very short timeout
	m, err := NewManager(tempDir, time.Millisecond)
	require.NoError(t, err)

	// Create sessions
	s1, err := m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)
	s2, err := m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)

	// Wait for expiry
	time.Sleep(10 * time.Millisecond)

	// Cleanup
	deleted, err := m.CleanupExpiredSessions(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, deleted)

	// Verify directories are gone
	_, err = os.Stat(m.storage.SessionDir(s1.ID))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(m.storage.SessionDir(s2.ID))
	assert.True(t, os.IsNotExist(err))
}

func TestSessionPersistence(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	// Create manager and session
	m1, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	session, err := m1.CreateSession(ctx, &types.CreateSessionRequest{
		Env: map[string]string{"KEY": "value"},
	})
	require.NoError(t, err)
	sessionID := session.ID

	// Create new manager (simulating restart)
	m2, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	// Session should still be accessible
	retrieved, err := m2.GetSession(ctx, sessionID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, sessionID, retrieved.ID)
	assert.Equal(t, "value", retrieved.Env["KEY"])
}

func TestTouchSession(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	m, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	// Create a session
	session, err := m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)

	// Save original timestamps (before the pointer gets modified)
	originalExpiry := session.ExpiresAt
	originalLastAccess := session.LastAccessAt

	// Wait a bit to ensure time difference
	time.Sleep(50 * time.Millisecond)

	// Touch the session
	err = m.TouchSession(ctx, session.ID)
	require.NoError(t, err)

	// Get updated session
	updated, err := m.GetSession(ctx, session.ID)
	require.NoError(t, err)

	// Expiry should be extended
	assert.True(t, updated.ExpiresAt.After(originalExpiry), "ExpiresAt should be after original")
	assert.True(t, updated.LastAccessAt.After(originalLastAccess), "LastAccessAt should be after original")
}

func TestStartCleanup(t *testing.T) {
	tempDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create manager with very short timeout
	m, err := NewManager(tempDir, 10*time.Millisecond)
	require.NoError(t, err)

	// Create sessions
	s1, err := m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)
	s2, err := m.CreateSession(ctx, &types.CreateSessionRequest{})
	require.NoError(t, err)

	// Track cleanup callback
	callbackCalled := make(chan struct{}, 1)
	callback := func(deleted int) {
		_ = deleted // Track that cleanup ran
		select {
		case callbackCalled <- struct{}{}:
		default:
		}
	}

	// Start cleanup with short interval
	m.StartCleanup(ctx, 20*time.Millisecond, callback)

	// Wait for sessions to expire and cleanup to run
	time.Sleep(50 * time.Millisecond)

	// Wait for callback
	select {
	case <-callbackCalled:
		// Callback was called
	case <-time.After(100 * time.Millisecond):
		t.Log("Timeout waiting for cleanup callback, checking directories anyway")
	}

	// Verify directories are gone (cleanup should have run)
	_, err = os.Stat(m.storage.SessionDir(s1.ID))
	assert.True(t, os.IsNotExist(err), "Session 1 directory should be deleted")
	_, err = os.Stat(m.storage.SessionDir(s2.ID))
	assert.True(t, os.IsNotExist(err), "Session 2 directory should be deleted")
}

func TestStartCleanupCancellation(t *testing.T) {
	tempDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())

	m, err := NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	cleanupCount := 0
	callback := func(deleted int) {
		cleanupCount++
	}

	// Start cleanup
	m.StartCleanup(ctx, 10*time.Millisecond, callback)

	// Let it run a bit
	time.Sleep(30 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait a bit
	time.Sleep(30 * time.Millisecond)

	// Get current count
	finalCount := cleanupCount

	// Wait more - count should not increase after cancellation
	time.Sleep(30 * time.Millisecond)

	assert.Equal(t, finalCount, cleanupCount, "Cleanup should stop after context cancellation")
}

func TestSessionTimeout(t *testing.T) {
	tempDir := t.TempDir()

	timeout := 5 * time.Hour
	m, err := NewManager(tempDir, timeout)
	require.NoError(t, err)

	assert.Equal(t, timeout, m.SessionTimeout())
}
