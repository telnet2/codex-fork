// Package api provides the HTTP API for the tool server.
package api

import (
	"context"

	"github.com/anthropics/codex-fork/gosdk/server/internal/session"
	"github.com/cloudwego/hertz/pkg/app/server"
)

// Server wraps the Hertz server with tool server dependencies.
type Server struct {
	hertz          *server.Hertz
	sessionManager *session.Manager
}

// Config holds server configuration.
type Config struct {
	Host           string
	Port           int
	TempDir        string
	SessionTimeout int // in hours
}

// NewServer creates a new API server.
func NewServer(cfg *Config, sessionManager *session.Manager) *Server {
	h := server.Default(server.WithHostPorts(cfg.Host + ":" + itoa(cfg.Port)))

	s := &Server{
		hertz:          h,
		sessionManager: sessionManager,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	// Health endpoints
	s.hertz.GET("/health", s.healthHandler)
	s.hertz.GET("/ready", s.readyHandler)

	// API v1
	v1 := s.hertz.Group("/api/v1")
	{
		// Session endpoints
		sessions := v1.Group("/sessions")
		{
			sessions.POST("", s.createSession)
			sessions.GET("/:id", s.getSession)
			sessions.DELETE("/:id", s.deleteSession)
			sessions.PUT("/:id/cwd", s.updateCwd)
		}

		// Tool endpoints (placeholder for Phase 2)
		tools := v1.Group("/tools")
		{
			tools.GET("", s.listTools)
			tools.POST("/:name/invoke", s.invokeTool)
		}

		// Execution endpoints (placeholder for Phase 2)
		executions := v1.Group("/executions")
		{
			executions.GET("/:id", s.getExecution)
			executions.GET("/:id/output", s.getExecutionOutput)
			executions.POST("/:id/cancel", s.cancelExecution)
		}
	}
}

// Run starts the server.
func (s *Server) Run() error {
	return s.hertz.Run()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.hertz.Shutdown(ctx)
}

// Hertz returns the underlying Hertz server for testing.
func (s *Server) Hertz() *server.Hertz {
	return s.hertz
}

// itoa converts int to string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
