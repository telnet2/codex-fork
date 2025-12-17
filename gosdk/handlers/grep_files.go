package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/anthropics/codex-fork/gosdk/tools"
)

const (
	// DefaultGrepLimit is the default limit for grep results
	DefaultGrepLimit = 100
	// MaxGrepLimit is the maximum limit for grep results
	MaxGrepLimit = 2000
	// GrepCommandTimeout is the timeout for the grep command
	GrepCommandTimeout = 30 * time.Second
)

// GrepFilesArgs represents the arguments for the grep_files tool
type GrepFilesArgs struct {
	Pattern string  `json:"pattern"`
	Include *string `json:"include,omitempty"`
	Path    *string `json:"path,omitempty"`
	Limit   int     `json:"limit"`
}

// GrepFilesHandler implements the grep_files tool
type GrepFilesHandler struct {
	// WorkingDir is the default working directory for searches
	WorkingDir string
}

// NewGrepFilesHandler creates a new GrepFilesHandler
func NewGrepFilesHandler(workingDir string) *GrepFilesHandler {
	return &GrepFilesHandler{
		WorkingDir: workingDir,
	}
}

// Name returns the tool name
func (h *GrepFilesHandler) Name() string {
	return "grep_files"
}

// Handle executes the grep_files tool
func (h *GrepFilesHandler) Handle(arguments string) (*tools.ToolOutput, error) {
	var args GrepFilesArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse function arguments: %w", err)
	}

	// Set defaults
	if args.Limit == 0 {
		args.Limit = DefaultGrepLimit
	}

	// Validate arguments
	pattern := strings.TrimSpace(args.Pattern)
	if pattern == "" {
		return tools.NewToolOutputError("pattern must not be empty"), nil
	}
	if args.Limit < 1 {
		return tools.NewToolOutputError("limit must be greater than zero"), nil
	}

	limit := args.Limit
	if limit > MaxGrepLimit {
		limit = MaxGrepLimit
	}

	// Determine search path
	searchPath := h.WorkingDir
	if args.Path != nil && *args.Path != "" {
		searchPath = *args.Path
	}

	// Verify path exists
	if _, err := os.Stat(searchPath); err != nil {
		return tools.NewToolOutputError(fmt.Sprintf("unable to access `%s`: %v", searchPath, err)), nil
	}

	// Parse include glob
	var include *string
	if args.Include != nil {
		trimmed := strings.TrimSpace(*args.Include)
		if trimmed != "" {
			include = &trimmed
		}
	}

	// Run ripgrep
	results, err := runRgSearch(pattern, include, searchPath, limit, h.WorkingDir)
	if err != nil {
		return tools.NewToolOutputError(err.Error()), nil
	}

	if len(results) == 0 {
		success := false
		return &tools.ToolOutput{
			Content: "No matches found.",
			Success: &success,
		}, nil
	}

	return tools.NewToolOutput(strings.Join(results, "\n")), nil
}

// runRgSearch runs ripgrep to search for files matching the pattern
func runRgSearch(pattern string, include *string, searchPath string, limit int, cwd string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GrepCommandTimeout)
	defer cancel()

	args := []string{
		"--files-with-matches",
		"--sortr=modified",
		"--regexp", pattern,
		"--no-messages",
	}

	if include != nil {
		args = append(args, "--glob", *include)
	}

	args = append(args, "--", searchPath)

	cmd := exec.CommandContext(ctx, "rg", args...)
	cmd.Dir = cwd

	output, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("rg timed out after %v", GrepCommandTimeout)
	}

	// Handle exit codes
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			switch exitErr.ExitCode() {
			case 1:
				// No matches found
				return nil, nil
			default:
				return nil, fmt.Errorf("rg failed: %s", string(exitErr.Stderr))
			}
		}
		return nil, fmt.Errorf("failed to launch rg: %v. Ensure ripgrep is installed and on PATH", err)
	}

	return parseGrepResults(output, limit), nil
}

// parseGrepResults parses ripgrep output
func parseGrepResults(output []byte, limit int) []string {
	var results []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		results = append(results, line)
		if len(results) >= limit {
			break
		}
	}
	return results
}

// IsRgAvailable checks if ripgrep is available on the system
func IsRgAvailable() bool {
	cmd := exec.Command("rg", "--version")
	return cmd.Run() == nil
}
