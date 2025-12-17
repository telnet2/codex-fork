package api

import (
	"context"

	"github.com/anthropics/codex-fork/gosdk/server/pkg/types"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// createSession handles POST /api/v1/sessions requests.
func (s *Server) createSession(ctx context.Context, c *app.RequestContext) {
	var req types.CreateSessionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	session, err := s.sessionManager.CreateSession(ctx, &req)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, types.NewAPIError(
			types.ErrorCodeInternalError,
			"Failed to create session: "+err.Error(),
		))
		return
	}

	c.JSON(consts.StatusCreated, types.CreateSessionResponse{
		SessionID:    session.ID,
		WorkspaceDir: session.WorkspaceDir,
		CreatedAt:    session.CreatedAt,
		ExpiresAt:    session.ExpiresAt,
	})
}

// getSession handles GET /api/v1/sessions/:id requests.
func (s *Server) getSession(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"Session ID is required",
		))
		return
	}

	session, err := s.sessionManager.GetSession(ctx, sessionID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, types.NewAPIError(
			types.ErrorCodeInternalError,
			"Failed to get session: "+err.Error(),
		))
		return
	}

	if session == nil {
		c.JSON(consts.StatusNotFound, types.NewAPIError(
			types.ErrorCodeSessionNotFound,
			"Session with ID '"+sessionID+"' not found",
		))
		return
	}

	c.JSON(consts.StatusOK, session)
}

// deleteSession handles DELETE /api/v1/sessions/:id requests.
func (s *Server) deleteSession(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"Session ID is required",
		))
		return
	}

	// Check if session exists
	session, err := s.sessionManager.GetSession(ctx, sessionID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, types.NewAPIError(
			types.ErrorCodeInternalError,
			"Failed to get session: "+err.Error(),
		))
		return
	}

	if session == nil {
		c.JSON(consts.StatusNotFound, types.NewAPIError(
			types.ErrorCodeSessionNotFound,
			"Session with ID '"+sessionID+"' not found",
		))
		return
	}

	if err := s.sessionManager.DeleteSession(ctx, sessionID); err != nil {
		c.JSON(consts.StatusInternalServerError, types.NewAPIError(
			types.ErrorCodeInternalError,
			"Failed to delete session: "+err.Error(),
		))
		return
	}

	c.Status(consts.StatusNoContent)
}

// updateCwd handles PUT /api/v1/sessions/:id/cwd requests.
func (s *Server) updateCwd(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"Session ID is required",
		))
		return
	}

	var req types.UpdateCwdRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	if req.Cwd == "" {
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			"cwd is required",
		))
		return
	}

	if err := s.sessionManager.UpdateCwd(ctx, sessionID, req.Cwd); err != nil {
		// Check if it's a "not found" error
		if err.Error() == "session not found" {
			c.JSON(consts.StatusNotFound, types.NewAPIError(
				types.ErrorCodeSessionNotFound,
				"Session with ID '"+sessionID+"' not found",
			))
			return
		}
		c.JSON(consts.StatusBadRequest, types.NewAPIError(
			types.ErrorCodeInvalidArguments,
			err.Error(),
		))
		return
	}

	// Return updated session
	session, err := s.sessionManager.GetSession(ctx, sessionID)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, types.NewAPIError(
			types.ErrorCodeInternalError,
			"Failed to get updated session: "+err.Error(),
		))
		return
	}

	c.JSON(consts.StatusOK, session)
}
