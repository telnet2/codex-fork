package api

import (
	"context"
	"strconv"

	"github.com/anthropics/codex-fork/gosdk/server/pkg/types"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// GetExecutionRequest contains query parameters for getting an execution.
type GetExecutionRequest struct {
	SessionID string `query:"session_id"`
}

// getExecution handles GET /api/v1/executions/:id requests.
func (s *Server) getExecution(ctx context.Context, c *app.RequestContext) {
	executionID := c.Param("id")
	if executionID == "" {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"Execution ID is required",
		))
		return
	}

	sessionID := string(c.QueryArgs().Peek("session_id"))
	if sessionID == "" {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"session_id query parameter is required",
		))
		return
	}

	exec, err := s.executor.GetExecution(sessionID, executionID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, types.NewAPIError(
			types.ErrorCodeInternalError,
			"Failed to get execution: "+err.Error(),
		))
		return
	}

	if exec == nil {
		c.JSON(consts.StatusNotFound, types.NewAPIError(
			types.ErrorCodeExecutionNotFound,
			"Execution not found: "+executionID,
		))
		return
	}

	c.JSON(consts.StatusOK, exec)
}

// getExecutionOutput handles GET /api/v1/executions/:id/output requests.
func (s *Server) getExecutionOutput(ctx context.Context, c *app.RequestContext) {
	executionID := c.Param("id")
	if executionID == "" {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"Execution ID is required",
		))
		return
	}

	sessionID := string(c.QueryArgs().Peek("session_id"))
	if sessionID == "" {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"session_id query parameter is required",
		))
		return
	}

	stream := string(c.QueryArgs().Peek("stream"))
	if stream == "" {
		stream = "stdout" // Default to stdout
	}
	if stream != "stdout" && stream != "stderr" {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"stream must be 'stdout' or 'stderr'",
		))
		return
	}

	// Parse offset and limit
	var offset, limit int64
	if offsetStr := string(c.QueryArgs().Peek("offset")); offsetStr != "" {
		var err error
		offset, err = strconv.ParseInt(offsetStr, 10, 64)
		if err != nil || offset < 0 {
			c.JSON(consts.StatusBadRequest, types.NewAPIError(
				types.ErrorCodeInvalidArguments,
				"offset must be a non-negative integer",
			))
			return
		}
	}

	if limitStr := string(c.QueryArgs().Peek("limit")); limitStr != "" {
		var err error
		limit, err = strconv.ParseInt(limitStr, 10, 64)
		if err != nil || limit < 0 {
			c.JSON(consts.StatusBadRequest, types.NewAPIError(
				types.ErrorCodeInvalidArguments,
				"limit must be a non-negative integer",
			))
			return
		}
	}

	output, err := s.executor.GetOutput(sessionID, executionID, stream, offset, limit)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, types.NewAPIError(
			types.ErrorCodeInternalError,
			"Failed to get output: "+err.Error(),
		))
		return
	}

	c.JSON(consts.StatusOK, output)
}

// cancelExecution handles POST /api/v1/executions/:id/cancel requests.
func (s *Server) cancelExecution(ctx context.Context, c *app.RequestContext) {
	executionID := c.Param("id")
	if executionID == "" {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"Execution ID is required",
		))
		return
	}

	if s.executor.Cancel(executionID) {
		c.JSON(consts.StatusOK, map[string]interface{}{
			"status":  "cancelled",
			"message": "Execution cancellation requested",
		})
	} else {
		c.JSON(consts.StatusNotFound, types.NewAPIError(
			types.ErrorCodeExecutionNotFound,
			"Execution not found or already completed: "+executionID,
		))
	}
}
