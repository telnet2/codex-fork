package handlers

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoRg(t *testing.T) {
	t.Helper()
	cmd := exec.Command("rg", "--version")
	if err := cmd.Run(); err != nil {
		t.Skip("ripgrep not available, skipping test")
	}
}

func TestGrepFilesHandler_BasicSearch(t *testing.T) {
	skipIfNoRg(t)

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "match_one.txt"), []byte("alpha beta gamma"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "match_two.txt"), []byte("alpha delta"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("omega"), 0644)
	require.NoError(t, err)

	handler := NewGrepFilesHandler(tmpDir)

	args := GrepFilesArgs{
		Pattern: "alpha",
		Limit:   10,
	}
	path := tmpDir
	args.Path = &path
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.True(t, *output.Success)
	assert.Contains(t, output.Content, "match_one.txt")
	assert.Contains(t, output.Content, "match_two.txt")
	assert.NotContains(t, output.Content, "other.txt")
}

func TestGrepFilesHandler_WithGlobFilter(t *testing.T) {
	skipIfNoRg(t)

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "match_one.rs"), []byte("alpha beta gamma"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "match_two.txt"), []byte("alpha delta"), 0644)
	require.NoError(t, err)

	handler := NewGrepFilesHandler(tmpDir)

	include := "*.rs"
	path := tmpDir
	args := GrepFilesArgs{
		Pattern: "alpha",
		Include: &include,
		Path:    &path,
		Limit:   10,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.True(t, *output.Success)
	assert.Contains(t, output.Content, "match_one.rs")
	assert.NotContains(t, output.Content, "match_two.txt")
}

func TestGrepFilesHandler_RespectsLimit(t *testing.T) {
	skipIfNoRg(t)

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "one.txt"), []byte("alpha one"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "two.txt"), []byte("alpha two"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "three.txt"), []byte("alpha three"), 0644)
	require.NoError(t, err)

	handler := NewGrepFilesHandler(tmpDir)

	path := tmpDir
	args := GrepFilesArgs{
		Pattern: "alpha",
		Path:    &path,
		Limit:   2,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)

	// Count lines in output
	lines := 0
	for _, c := range output.Content {
		if c == '\n' {
			lines++
		}
	}
	// Should have at most 2 matches (plus potential newline at end)
	assert.LessOrEqual(t, lines, 2)
}

func TestGrepFilesHandler_NoMatches(t *testing.T) {
	skipIfNoRg(t)

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "one.txt"), []byte("omega"), 0644)
	require.NoError(t, err)

	handler := NewGrepFilesHandler(tmpDir)

	path := tmpDir
	args := GrepFilesArgs{
		Pattern: "alpha",
		Path:    &path,
		Limit:   5,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Equal(t, "No matches found.", output.Content)
	assert.False(t, *output.Success)
}

func TestGrepFilesHandler_EmptyPattern(t *testing.T) {
	handler := NewGrepFilesHandler("/tmp")

	args := GrepFilesArgs{
		Pattern: "",
		Limit:   10,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Contains(t, output.Content, "pattern must not be empty")
}

func TestGrepFilesHandler_InvalidPath(t *testing.T) {
	skipIfNoRg(t)

	handler := NewGrepFilesHandler("/tmp")

	path := "/nonexistent/path/12345"
	args := GrepFilesArgs{
		Pattern: "test",
		Path:    &path,
		Limit:   10,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Contains(t, output.Content, "unable to access")
}

func TestGrepFilesHandler_WhitespacePattern(t *testing.T) {
	handler := NewGrepFilesHandler("/tmp")

	args := GrepFilesArgs{
		Pattern: "   ",
		Limit:   10,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Contains(t, output.Content, "pattern must not be empty")
}

func TestParseGrepResults(t *testing.T) {
	output := []byte("/tmp/file_a.rs\n/tmp/file_b.rs\n")
	results := parseGrepResults(output, 10)
	assert.Equal(t, []string{"/tmp/file_a.rs", "/tmp/file_b.rs"}, results)
}

func TestParseGrepResults_TruncatesAfterLimit(t *testing.T) {
	output := []byte("/tmp/file_a.rs\n/tmp/file_b.rs\n/tmp/file_c.rs\n")
	results := parseGrepResults(output, 2)
	assert.Len(t, results, 2)
	assert.Equal(t, []string{"/tmp/file_a.rs", "/tmp/file_b.rs"}, results)
}

func TestParseGrepResults_HandlesEmptyLines(t *testing.T) {
	output := []byte("/tmp/file_a.rs\n\n/tmp/file_b.rs\n\n")
	results := parseGrepResults(output, 10)
	assert.Equal(t, []string{"/tmp/file_a.rs", "/tmp/file_b.rs"}, results)
}

func TestIsRgAvailable(t *testing.T) {
	// This test just ensures the function doesn't panic
	_ = IsRgAvailable()
}
