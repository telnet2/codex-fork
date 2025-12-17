package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/anthropics/codex-fork/gosdk/server/internal/executor"
	"github.com/anthropics/codex-fork/gosdk/server/internal/session"
	"github.com/anthropics/codex-fork/gosdk/server/pkg/types"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) *Server {
	tempDir := t.TempDir()

	sessionManager, err := session.NewManager(tempDir, time.Hour)
	require.NoError(t, err)

	exec := executor.NewExecutor(sessionManager.Storage())

	cfg := &Config{
		Host:           "127.0.0.1",
		Port:           0, // Random port
		TempDir:        tempDir,
		SessionTimeout: 1,
	}

	return NewServer(cfg, sessionManager, exec)
}

func TestHealthEndpoint(t *testing.T) {
	server := setupTestServer(t)

	w := ut.PerformRequest(server.Hertz().Engine, http.MethodGet, "/health", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestReadyEndpoint(t *testing.T) {
	server := setupTestServer(t)

	w := ut.PerformRequest(server.Hertz().Engine, http.MethodGet, "/ready", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "ready", resp.Status)
}

func TestCreateSessionEndpoint(t *testing.T) {
	server := setupTestServer(t)

	// Create session with empty body
	body := bytes.NewBufferString("{}")
	w := ut.PerformRequest(server.Hertz().Engine, http.MethodPost, "/api/v1/sessions",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp types.CreateSessionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Len(t, resp.SessionID, 26) // ULID length
	assert.NotEmpty(t, resp.WorkspaceDir)
	assert.False(t, resp.CreatedAt.IsZero())
	assert.False(t, resp.ExpiresAt.IsZero())
}

func TestCreateSessionWithCwd(t *testing.T) {
	server := setupTestServer(t)

	cwd := t.TempDir()
	reqBody := types.CreateSessionRequest{Cwd: cwd}
	bodyBytes, _ := json.Marshal(reqBody)
	body := bytes.NewBuffer(bodyBytes)

	w := ut.PerformRequest(server.Hertz().Engine, http.MethodPost, "/api/v1/sessions",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp types.CreateSessionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.SessionID, 26)
}

func TestGetSessionEndpoint(t *testing.T) {
	server := setupTestServer(t)

	// First create a session
	body := bytes.NewBufferString("{}")
	w := ut.PerformRequest(server.Hertz().Engine, http.MethodPost, "/api/v1/sessions",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp types.CreateSessionResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	// Now get the session
	w = ut.PerformRequest(server.Hertz().Engine, http.MethodGet,
		"/api/v1/sessions/"+createResp.SessionID, nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var session types.Session
	err = json.Unmarshal(w.Body.Bytes(), &session)
	require.NoError(t, err)
	assert.Equal(t, createResp.SessionID, session.ID)
}

func TestGetNonExistentSessionEndpoint(t *testing.T) {
	server := setupTestServer(t)

	w := ut.PerformRequest(server.Hertz().Engine, http.MethodGet,
		"/api/v1/sessions/nonexistent", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp types.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, types.ErrorCodeSessionNotFound, errResp.Error.Code)
}

func TestDeleteSessionEndpoint(t *testing.T) {
	server := setupTestServer(t)

	// First create a session
	body := bytes.NewBufferString("{}")
	w := ut.PerformRequest(server.Hertz().Engine, http.MethodPost, "/api/v1/sessions",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp types.CreateSessionResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	// Delete the session
	w = ut.PerformRequest(server.Hertz().Engine, http.MethodDelete,
		"/api/v1/sessions/"+createResp.SessionID, nil)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify it's gone
	w = ut.PerformRequest(server.Hertz().Engine, http.MethodGet,
		"/api/v1/sessions/"+createResp.SessionID, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateCwdEndpoint(t *testing.T) {
	server := setupTestServer(t)

	// First create a session
	body := bytes.NewBufferString("{}")
	w := ut.PerformRequest(server.Hertz().Engine, http.MethodPost, "/api/v1/sessions",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp types.CreateSessionResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	// Update cwd
	newCwd := t.TempDir()
	updateReq := types.UpdateCwdRequest{Cwd: newCwd}
	bodyBytes, _ := json.Marshal(updateReq)
	body = bytes.NewBuffer(bodyBytes)

	w = ut.PerformRequest(server.Hertz().Engine, http.MethodPut,
		"/api/v1/sessions/"+createResp.SessionID+"/cwd",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	assert.Equal(t, http.StatusOK, w.Code)

	var session types.Session
	err = json.Unmarshal(w.Body.Bytes(), &session)
	require.NoError(t, err)
	assert.Equal(t, newCwd, session.Cwd)
}

func TestUpdateCwdInvalidPath(t *testing.T) {
	server := setupTestServer(t)

	// First create a session
	body := bytes.NewBufferString("{}")
	w := ut.PerformRequest(server.Hertz().Engine, http.MethodPost, "/api/v1/sessions",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp types.CreateSessionResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	// Try to update cwd to invalid path
	updateReq := types.UpdateCwdRequest{Cwd: "/nonexistent/path/that/does/not/exist"}
	bodyBytes, _ := json.Marshal(updateReq)
	body = bytes.NewBuffer(bodyBytes)

	w = ut.PerformRequest(server.Hertz().Engine, http.MethodPut,
		"/api/v1/sessions/"+createResp.SessionID+"/cwd",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListToolsEndpoint(t *testing.T) {
	server := setupTestServer(t)

	w := ut.PerformRequest(server.Hertz().Engine, http.MethodGet,
		"/api/v1/tools", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp ListToolsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	// Should have registered tools
	assert.NotEmpty(t, resp.Tools)
	// Verify some known tools exist
	toolNames := make(map[string]bool)
	for _, tool := range resp.Tools {
		toolNames[tool.Name] = true
	}
	assert.True(t, toolNames["read_file"], "read_file tool should be registered")
	assert.True(t, toolNames["list_dir"], "list_dir tool should be registered")
}

func TestInvokeReadFileTool(t *testing.T) {
	server := setupTestServer(t)

	// Create a test file
	tempFile := t.TempDir() + "/test.txt"
	err := os.WriteFile(tempFile, []byte("Hello, World!"), 0644)
	require.NoError(t, err)

	// Invoke read_file tool
	body := bytes.NewBufferString(`{"arguments": {"file_path": "` + tempFile + `"}}`)
	w := ut.PerformRequest(server.Hertz().Engine, http.MethodPost,
		"/api/v1/tools/read_file/invoke",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp types.InvokeToolResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.SessionID)
	assert.NotEmpty(t, resp.ExecutionID)
	assert.NotNil(t, resp.Execution)
	assert.Equal(t, types.ExecutionStatusCompleted, resp.Execution.Status)
}

func TestInvokeUnknownTool(t *testing.T) {
	server := setupTestServer(t)

	body := bytes.NewBufferString(`{"arguments": {}}`)
	w := ut.PerformRequest(server.Hertz().Engine, http.MethodPost,
		"/api/v1/tools/nonexistent_tool/invoke",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp types.InvokeToolResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Execution)
	assert.Equal(t, types.ExecutionStatusFailed, resp.Execution.Status)
	assert.Contains(t, resp.Execution.Error, "unknown tool")
}

func TestConcurrentSessionCreation(t *testing.T) {
	server := setupTestServer(t)
	ctx := context.Background()
	_ = ctx

	// Create multiple sessions concurrently
	const numSessions = 10
	results := make(chan string, numSessions)
	errors := make(chan error, numSessions)

	for i := 0; i < numSessions; i++ {
		go func() {
			body := bytes.NewBufferString("{}")
			w := ut.PerformRequest(server.Hertz().Engine, http.MethodPost, "/api/v1/sessions",
				&ut.Body{Body: body, Len: body.Len()},
				ut.Header{Key: "Content-Type", Value: "application/json"})

			if w.Code != http.StatusCreated {
				errors <- assert.AnError
				return
			}

			var resp types.CreateSessionResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				errors <- err
				return
			}
			results <- resp.SessionID
		}()
	}

	// Collect results
	ids := make(map[string]bool)
	for i := 0; i < numSessions; i++ {
		select {
		case id := <-results:
			assert.False(t, ids[id], "Session IDs should be unique")
			ids[id] = true
		case err := <-errors:
			t.Fatalf("Concurrent session creation failed: %v", err)
		}
	}

	assert.Len(t, ids, numSessions)
}
