package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFileHandler_SliceMode(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := "alpha\nbeta\ngamma\n"
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	handler := NewReadFileHandler()

	args := ReadFileArgs{
		FilePath: tmpFile,
		Offset:   2,
		Limit:    2,
		Mode:     ReadModeSlice,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Contains(t, output.Content, "L2: beta")
	assert.Contains(t, output.Content, "L3: gamma")
}

func TestReadFileHandler_OffsetExceedsLength(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(tmpFile, []byte("only\n"), 0644)
	require.NoError(t, err)

	handler := NewReadFileHandler()

	args := ReadFileArgs{
		FilePath: tmpFile,
		Offset:   3,
		Limit:    1,
		Mode:     ReadModeSlice,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Contains(t, output.Content, "offset exceeds file length")
	assert.False(t, *output.Success)
}

func TestReadFileHandler_NonUtf8Lines(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	// Write non-UTF8 bytes followed by plain text
	content := []byte{0xff, 0xfe, '\n', 'p', 'l', 'a', 'i', 'n', '\n'}
	err := os.WriteFile(tmpFile, content, 0644)
	require.NoError(t, err)

	handler := NewReadFileHandler()

	args := ReadFileArgs{
		FilePath: tmpFile,
		Offset:   1,
		Limit:    2,
		Mode:     ReadModeSlice,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.True(t, *output.Success)
	assert.Contains(t, output.Content, "L2: plain")
}

func TestReadFileHandler_RespectsLimit(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := "first\nsecond\nthird\n"
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	handler := NewReadFileHandler()

	args := ReadFileArgs{
		FilePath: tmpFile,
		Offset:   1,
		Limit:    2,
		Mode:     ReadModeSlice,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Contains(t, output.Content, "L1: first")
	assert.Contains(t, output.Content, "L2: second")
	assert.NotContains(t, output.Content, "L3: third")
}

func TestReadFileHandler_TruncatesLongLines(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	// Create a line longer than MaxLineLength
	longLine := make([]byte, MaxLineLength+50)
	for i := range longLine {
		longLine[i] = 'x'
	}
	longLine = append(longLine, '\n')
	err := os.WriteFile(tmpFile, longLine, 0644)
	require.NoError(t, err)

	handler := NewReadFileHandler()

	args := ReadFileArgs{
		FilePath: tmpFile,
		Offset:   1,
		Limit:    1,
		Mode:     ReadModeSlice,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	// The line should be truncated
	assert.LessOrEqual(t, len(output.Content), MaxLineLength+20) // +20 for "L1: " prefix
}

func TestReadFileHandler_NegativeOffset(t *testing.T) {
	// Create a temp file to avoid file-not-found error
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(tmpFile, []byte("content\n"), 0644)
	require.NoError(t, err)

	handler := NewReadFileHandler()

	// Use raw JSON to set negative offset (bypasses default handling)
	argsJSON := `{"file_path":"` + tmpFile + `","offset":-1,"limit":10}`

	output, err := handler.Handle(argsJSON)
	require.NoError(t, err)
	assert.Contains(t, output.Content, "offset must be a 1-indexed line number")
}

func TestReadFileHandler_NegativeLimit(t *testing.T) {
	// Create a temp file to avoid file-not-found error
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(tmpFile, []byte("content\n"), 0644)
	require.NoError(t, err)

	handler := NewReadFileHandler()

	// Use raw JSON to set negative limit (bypasses default handling)
	argsJSON := `{"file_path":"` + tmpFile + `","offset":1,"limit":-1}`

	output, err := handler.Handle(argsJSON)
	require.NoError(t, err)
	assert.Contains(t, output.Content, "limit must be greater than zero")
}

func TestReadFileHandler_RelativePath(t *testing.T) {
	handler := NewReadFileHandler()

	args := ReadFileArgs{
		FilePath: "relative/path.txt",
		Offset:   1,
		Limit:    10,
		Mode:     ReadModeSlice,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Contains(t, output.Content, "file_path must be an absolute path")
}

func TestReadFileHandler_IndentationMode(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := `fn outer() {
    if cond {
        inner();
    }
    tail();
}
`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	handler := NewReadFileHandler()

	anchorLine := 3
	maxLevels := 1
	args := ReadFileArgs{
		FilePath: tmpFile,
		Offset:   3,
		Limit:    10,
		Mode:     ReadModeIndentation,
		Indentation: &IndentationArgs{
			AnchorLine:      &anchorLine,
			MaxLevels:       maxLevels,
			IncludeSiblings: false,
			IncludeHeader:   true,
		},
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.True(t, *output.Success)
	assert.Contains(t, output.Content, "if cond")
	assert.Contains(t, output.Content, "inner()")
}

func TestReadFileHandler_IndentationMode_WithSiblings(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := `fn wrapper() {
    if first {
        do_first();
    }
    if second {
        do_second();
    }
}
`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	handler := NewReadFileHandler()

	anchorLine := 3
	maxLevels := 1
	args := ReadFileArgs{
		FilePath: tmpFile,
		Offset:   3,
		Limit:    50,
		Mode:     ReadModeIndentation,
		Indentation: &IndentationArgs{
			AnchorLine:      &anchorLine,
			MaxLevels:       maxLevels,
			IncludeSiblings: true,
			IncludeHeader:   true,
		},
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.True(t, *output.Success)
	assert.Contains(t, output.Content, "if first")
	assert.Contains(t, output.Content, "if second")
}

func TestMeasureIndent(t *testing.T) {
	tests := []struct {
		line     string
		expected int
	}{
		{"no indent", 0},
		{"  two spaces", 2},
		{"    four spaces", 4},
		{"\ttab", TabWidth},
		{"\t\ttwo tabs", TabWidth * 2},
		{"  \t  mixed", 2 + TabWidth + 2},
	}

	for _, tc := range tests {
		t.Run(tc.line, func(t *testing.T) {
			result := measureIndent(tc.line)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLineRecord_IsBlank(t *testing.T) {
	blank := &lineRecord{Raw: "   "}
	assert.True(t, blank.IsBlank())

	notBlank := &lineRecord{Raw: "  hello  "}
	assert.False(t, notBlank.IsBlank())
}

func TestLineRecord_IsComment(t *testing.T) {
	tests := []struct {
		raw      string
		expected bool
	}{
		{"# comment", true},
		{"// comment", true},
		{"-- comment", true},
		{"  # indented comment", true},
		{"not a comment", false},
		{"code # with comment", false}, // Comment not at start
	}

	for _, tc := range tests {
		t.Run(tc.raw, func(t *testing.T) {
			record := &lineRecord{Raw: tc.raw}
			assert.Equal(t, tc.expected, record.IsComment())
		})
	}
}

func TestFormatLine(t *testing.T) {
	short := "short line"
	assert.Equal(t, short, formatLine(short))

	long := make([]byte, MaxLineLength+100)
	for i := range long {
		long[i] = 'x'
	}
	result := formatLine(string(long))
	assert.Equal(t, MaxLineLength, len(result))
}
