package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: SSE streaming tests are limited with the ut.PerformRequest mock framework.
// Full SSE streaming tests would require integration tests with a real HTTP client.

func TestStreamingEndpointBadRequest(t *testing.T) {
	server := setupTestServer(t)

	// Test with invalid JSON - this should fail before SSE streaming starts
	body := bytes.NewBufferString(`{invalid json}`)
	w := ut.PerformRequest(server.Hertz().Engine, http.MethodPost,
		"/api/v1/tools/read_file/invoke/stream",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStreamingEndpointSessionNotFound(t *testing.T) {
	server := setupTestServer(t)

	// Invoke streaming endpoint with non-existent session - this should fail before SSE streaming starts
	body := bytes.NewBufferString(`{"session_id": "nonexistent", "arguments": {"file_path": "/tmp/test.txt"}}`)
	w := ut.PerformRequest(server.Hertz().Engine, http.MethodPost,
		"/api/v1/tools/read_file/invoke/stream",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestStreamingEndpointRouteExists(t *testing.T) {
	server := setupTestServer(t)

	// First create a session
	body := bytes.NewBufferString("{}")
	w := ut.PerformRequest(server.Hertz().Engine, http.MethodPost, "/api/v1/sessions",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	require.Equal(t, http.StatusCreated, w.Code)

	// Parse session ID from response
	var createResp struct {
		SessionID string `json:"session_id"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	// Verify the streaming route is registered by checking it doesn't return 404
	// Note: We can't fully test SSE with ut.PerformRequest, but we can verify the route exists
	body = bytes.NewBufferString(`{"session_id": "` + createResp.SessionID + `", "arguments": {}}`)
	w = ut.PerformRequest(server.Hertz().Engine, http.MethodPost,
		"/api/v1/tools/read_file/invoke/stream",
		&ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	// Route should exist (not 404), though it may fail with SSE in mock mode
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}
