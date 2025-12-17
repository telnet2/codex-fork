// Package main provides the entry point for the tool server.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/anthropics/codex-fork/gosdk/server/internal/api"
	"github.com/anthropics/codex-fork/gosdk/server/internal/executor"
	"github.com/anthropics/codex-fork/gosdk/server/internal/session"
)

func main() {
	// Parse command line flags
	host := flag.String("host", "0.0.0.0", "Host to listen on")
	port := flag.Int("port", 8080, "Port to listen on")
	tempDir := flag.String("temp-dir", "", "Base temporary directory (default: system temp)")
	sessionTimeout := flag.Int("session-timeout", 72, "Session timeout in hours")
	cleanupInterval := flag.Int("cleanup-interval", 60, "Cleanup interval in minutes")
	flag.Parse()

	// Determine temp directory
	baseDir := *tempDir
	if baseDir == "" {
		baseDir = filepath.Join(os.TempDir(), "tool-server")
	}

	log.Printf("Starting tool server on %s:%d", *host, *port)
	log.Printf("Using base directory: %s", baseDir)
	log.Printf("Session timeout: %d hours", *sessionTimeout)
	log.Printf("Cleanup interval: %d minutes", *cleanupInterval)

	// Create session manager
	sessionManager, err := session.NewManager(baseDir, time.Duration(*sessionTimeout)*time.Hour)
	if err != nil {
		log.Fatalf("Failed to create session manager: %v", err)
	}

	// Create executor with tool registry
	exec := executor.NewExecutor(sessionManager.Storage())
	log.Printf("Registered %d tools", len(exec.GetRegistry().GetTools()))

	// Create and configure server
	cfg := &api.Config{
		Host:           *host,
		Port:           *port,
		TempDir:        baseDir,
		SessionTimeout: *sessionTimeout,
	}

	server := api.NewServer(cfg, sessionManager, exec)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start session cleanup background job
	sessionManager.StartCleanup(ctx, time.Duration(*cleanupInterval)*time.Minute, func(deleted int) {
		if deleted > 0 {
			log.Printf("Cleaned up %d expired sessions", deleted)
		}
	})
	log.Printf("Started session cleanup background job")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	// Start server
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
