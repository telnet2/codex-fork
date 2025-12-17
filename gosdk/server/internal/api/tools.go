package api

import (
	"context"
	"encoding/json"

	"github.com/anthropics/codex-fork/gosdk/schema"
	sseutil "github.com/anthropics/codex-fork/gosdk/server/internal/sse"
	"github.com/anthropics/codex-fork/gosdk/server/pkg/types"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// ToolInfo represents basic tool information.
type ToolInfo struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Parameters  *schema.JSONSchema `json:"parameters,omitempty"`
}

// ListToolsResponse is the response for listing tools.
type ListToolsResponse struct {
	Tools []ToolInfo `json:"tools"`
}

// listTools handles GET /api/v1/tools requests.
func (s *Server) listTools(ctx context.Context, c *app.RequestContext) {
	registry := s.executor.GetRegistry()
	specs := registry.GetTools()

	toolList := make([]ToolInfo, 0, len(specs))
	for _, spec := range specs {
		toolList = append(toolList, ToolInfo{
			Name:        spec.GetName(),
			Description: spec.Description,
			Parameters:  spec.Parameters,
		})
	}

	c.JSON(consts.StatusOK, ListToolsResponse{
		Tools: toolList,
	})
}

// InvokeRequest is the request body for tool invocation.
type InvokeRequest struct {
	SessionID string          `json:"session_id,omitempty"`
	Arguments json.RawMessage `json:"arguments"`
}

// invokeTool handles POST /api/v1/tools/:name/invoke requests.
func (s *Server) invokeTool(ctx context.Context, c *app.RequestContext) {
	toolName := c.Param("name")
	if toolName == "" {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"Tool name is required",
		))
		return
	}

	var req InvokeRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	// Get or create session
	var sess *types.Session
	var err error

	if req.SessionID != "" {
		sess, err = s.sessionManager.GetSession(ctx, req.SessionID)
		if err != nil {
			c.JSON(consts.StatusInternalServerError, types.NewAPIError(
				types.ErrorCodeInternalError,
				"Failed to get session: "+err.Error(),
			))
			return
		}
		if sess == nil {
			c.JSON(consts.StatusNotFound, types.NewAPIError(
				types.ErrorCodeSessionNotFound,
				"Session not found: "+req.SessionID,
			))
			return
		}
		// Touch session to extend expiry
		s.sessionManager.TouchSession(ctx, req.SessionID)
	} else {
		// Create new session
		sess, err = s.sessionManager.CreateSession(ctx, &types.CreateSessionRequest{})
		if err != nil {
			c.JSON(consts.StatusInternalServerError, types.NewAPIError(
				types.ErrorCodeInternalError,
				"Failed to create session: "+err.Error(),
			))
			return
		}
	}

	// Create invocation request
	invokeReq := &types.InvokeToolRequest{
		SessionID: sess.ID,
		ToolName:  toolName,
		Arguments: req.Arguments,
	}

	// Execute the tool
	exec, err := s.executor.Execute(ctx, sess, invokeReq)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, types.NewAPIError(
			types.ErrorCodeExecutionFailed,
			"Failed to execute tool: "+err.Error(),
		))
		return
	}

	c.JSON(consts.StatusOK, types.InvokeToolResponse{
		SessionID:   sess.ID,
		ExecutionID: exec.ID,
		Execution:   exec,
	})
}

// invokeToolStream handles POST /api/v1/tools/:name/invoke/stream requests.
// This endpoint uses Server-Sent Events to stream tool execution output in real-time.
func (s *Server) invokeToolStream(ctx context.Context, c *app.RequestContext) {
	toolName := c.Param("name")
	if toolName == "" {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"Tool name is required",
		))
		return
	}

	var req InvokeRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	// Get or create session
	var sess *types.Session
	var err error

	if req.SessionID != "" {
		sess, err = s.sessionManager.GetSession(ctx, req.SessionID)
		if err != nil {
			c.JSON(consts.StatusInternalServerError, types.NewAPIError(
				types.ErrorCodeInternalError,
				"Failed to get session: "+err.Error(),
			))
			return
		}
		if sess == nil {
			c.JSON(consts.StatusNotFound, types.NewAPIError(
				types.ErrorCodeSessionNotFound,
				"Session not found: "+req.SessionID,
			))
			return
		}
		// Touch session to extend expiry
		s.sessionManager.TouchSession(ctx, req.SessionID)
	} else {
		// Create new session
		sess, err = s.sessionManager.CreateSession(ctx, &types.CreateSessionRequest{})
		if err != nil {
			c.JSON(consts.StatusInternalServerError, types.NewAPIError(
				types.ErrorCodeInternalError,
				"Failed to create session: "+err.Error(),
			))
			return
		}
	}

	// Create invocation request
	invokeReq := &types.InvokeToolRequest{
		SessionID: sess.ID,
		ToolName:  toolName,
		Arguments: req.Arguments,
	}

	// Set up SSE streaming
	c.SetContentType("text/event-stream")
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.Header.Set("X-Accel-Buffering", "no")

	// Create SSE streamer
	streamer := sseutil.NewStreamer(ctx, c)

	// Execute the tool with streaming
	_, err = s.executor.ExecuteStreaming(ctx, sess, invokeReq, streamer)
	if err != nil {
		// Try to send error via SSE if possible
		streamer.SendError(&sseutil.ErrorEvent{
			ExecutionID: "",
			Error:       "Failed to execute tool: " + err.Error(),
			DurationMs:  0,
		})
	}
}
