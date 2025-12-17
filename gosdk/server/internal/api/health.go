package api

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status string `json:"status"`
}

// healthHandler handles GET /health requests.
func (s *Server) healthHandler(ctx context.Context, c *app.RequestContext) {
	c.JSON(consts.StatusOK, HealthResponse{Status: "ok"})
}

// readyHandler handles GET /ready requests.
func (s *Server) readyHandler(ctx context.Context, c *app.RequestContext) {
	// Could add more sophisticated readiness checks here
	c.JSON(consts.StatusOK, HealthResponse{Status: "ready"})
}
