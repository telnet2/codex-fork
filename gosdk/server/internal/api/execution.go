package api

import (
	"context"

	"github.com/anthropics/codex-fork/gosdk/server/pkg/types"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// getExecution handles GET /api/v1/executions/:id requests.
// TODO: Implement in Phase 2.
func (s *Server) getExecution(ctx context.Context, c *app.RequestContext) {
	// Placeholder - will be implemented in Phase 2
	c.JSON(consts.StatusNotImplemented, types.NewAPIError(
		types.ErrorCodeInternalError,
		"Execution retrieval not yet implemented",
	))
}

// getExecutionOutput handles GET /api/v1/executions/:id/output requests.
// TODO: Implement in Phase 2.
func (s *Server) getExecutionOutput(ctx context.Context, c *app.RequestContext) {
	// Placeholder - will be implemented in Phase 2
	c.JSON(consts.StatusNotImplemented, types.NewAPIError(
		types.ErrorCodeInternalError,
		"Execution output retrieval not yet implemented",
	))
}

// cancelExecution handles POST /api/v1/executions/:id/cancel requests.
// TODO: Implement in Phase 4.
func (s *Server) cancelExecution(ctx context.Context, c *app.RequestContext) {
	// Placeholder - will be implemented in Phase 4
	c.JSON(consts.StatusNotImplemented, types.NewAPIError(
		types.ErrorCodeInternalError,
		"Execution cancellation not yet implemented",
	))
}
