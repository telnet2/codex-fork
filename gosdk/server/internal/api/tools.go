package api

import (
	"context"

	"github.com/anthropics/codex-fork/gosdk/server/pkg/types"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// ToolInfo represents basic tool information.
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ListToolsResponse is the response for listing tools.
type ListToolsResponse struct {
	Tools []ToolInfo `json:"tools"`
}

// listTools handles GET /api/v1/tools requests.
// TODO: Implement in Phase 2 with actual tool registry.
func (s *Server) listTools(ctx context.Context, c *app.RequestContext) {
	// Placeholder - will be implemented in Phase 2
	c.JSON(consts.StatusOK, ListToolsResponse{
		Tools: []ToolInfo{},
	})
}

// invokeTool handles POST /api/v1/tools/:name/invoke requests.
// TODO: Implement in Phase 2 with actual tool execution.
func (s *Server) invokeTool(ctx context.Context, c *app.RequestContext) {
	// Placeholder - will be implemented in Phase 2
	c.JSON(consts.StatusNotImplemented, types.NewAPIError(
		types.ErrorCodeInternalError,
		"Tool invocation not yet implemented",
	))
}
