package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListDirHandler_BasicListing(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files and directories
	err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644)
	require.NoError(t, err)
	err = os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	require.NoError(t, err)

	handler := NewListDirHandler()

	args := ListDirArgs{
		DirPath: tmpDir,
		Offset:  1,
		Limit:   20,
		Depth:   1,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.True(t, *output.Success)
	assert.Contains(t, output.Content, "file.txt")
	assert.Contains(t, output.Content, "subdir/")
}

func TestListDirHandler_RespectsDepth(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directories
	nested := filepath.Join(tmpDir, "nested")
	deeper := filepath.Join(nested, "deeper")
	err := os.MkdirAll(deeper, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("root"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(nested, "child.txt"), []byte("child"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(deeper, "grandchild.txt"), []byte("deep"), 0644)
	require.NoError(t, err)

	handler := NewListDirHandler()

	// Depth 1 - only top level
	args := ListDirArgs{
		DirPath: tmpDir,
		Offset:  1,
		Limit:   10,
		Depth:   1,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Contains(t, output.Content, "nested/")
	assert.Contains(t, output.Content, "root.txt")
	assert.NotContains(t, output.Content, "child.txt")
	assert.NotContains(t, output.Content, "grandchild.txt")

	// Depth 2 - include nested
	args.Depth = 2
	argsJSON, _ = json.Marshal(args)

	output, err = handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Contains(t, output.Content, "child.txt")
	assert.Contains(t, output.Content, "deeper/")
	assert.NotContains(t, output.Content, "grandchild.txt")

	// Depth 3 - include all
	args.Depth = 3
	argsJSON, _ = json.Marshal(args)

	output, err = handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Contains(t, output.Content, "grandchild.txt")
}

func TestListDirHandler_OffsetExceedsEntries(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	require.NoError(t, err)

	handler := NewListDirHandler()

	args := ListDirArgs{
		DirPath: tmpDir,
		Offset:  10,
		Limit:   1,
		Depth:   1,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Contains(t, output.Content, "offset exceeds directory entry count")
}

func TestListDirHandler_NegativeOffset(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	handler := NewListDirHandler()

	// Use raw JSON to set negative offset
	argsJSON := `{"dir_path":"` + tmpDir + `","offset":-1,"limit":10,"depth":1}`

	output, err := handler.Handle(argsJSON)
	require.NoError(t, err)
	assert.Contains(t, output.Content, "offset must be a 1-indexed entry number")
}

func TestListDirHandler_NegativeLimit(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	handler := NewListDirHandler()

	// Use raw JSON to set negative limit
	argsJSON := `{"dir_path":"` + tmpDir + `","offset":1,"limit":-1,"depth":1}`

	output, err := handler.Handle(argsJSON)
	require.NoError(t, err)
	assert.Contains(t, output.Content, "limit must be greater than zero")
}

func TestListDirHandler_NegativeDepth(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	handler := NewListDirHandler()

	// Use raw JSON to set negative depth
	argsJSON := `{"dir_path":"` + tmpDir + `","offset":1,"limit":10,"depth":-1}`

	output, err := handler.Handle(argsJSON)
	require.NoError(t, err)
	assert.Contains(t, output.Content, "depth must be greater than zero")
}

func TestListDirHandler_RelativePath(t *testing.T) {
	handler := NewListDirHandler()

	args := ListDirArgs{
		DirPath: "relative/path",
		Offset:  1,
		Limit:   10,
		Depth:   1,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Contains(t, output.Content, "dir_path must be an absolute path")
}

func TestListDirHandler_TruncatesResults(t *testing.T) {
	tmpDir := t.TempDir()

	// Create many files (more than limit)
	for i := 0; i < 40; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("file_%02d.txt", i))
		err := os.WriteFile(filename, []byte("content"), 0644)
		require.NoError(t, err)
	}

	handler := NewListDirHandler()

	args := ListDirArgs{
		DirPath: tmpDir,
		Offset:  1,
		Limit:   25,
		Depth:   1,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Contains(t, output.Content, "More than")
}

func TestListDirHandler_AbsolutePathOutput(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	handler := NewListDirHandler()

	args := ListDirArgs{
		DirPath: tmpDir,
		Offset:  1,
		Limit:   10,
		Depth:   1,
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.Contains(t, output.Content, "Absolute path: "+tmpDir)
}

func TestFormatEntryLine(t *testing.T) {
	tests := []struct {
		entry    DirEntry
		expected string
	}{
		{
			entry:    DirEntry{DisplayName: "file.txt", Kind: KindFile, Depth: 0},
			expected: "file.txt",
		},
		{
			entry:    DirEntry{DisplayName: "dir", Kind: KindDirectory, Depth: 0},
			expected: "dir/",
		},
		{
			entry:    DirEntry{DisplayName: "link", Kind: KindSymlink, Depth: 0},
			expected: "link@",
		},
		{
			entry:    DirEntry{DisplayName: "other", Kind: KindOther, Depth: 0},
			expected: "other?",
		},
		{
			entry:    DirEntry{DisplayName: "nested", Kind: KindFile, Depth: 1},
			expected: "  nested",
		},
		{
			entry:    DirEntry{DisplayName: "deep", Kind: KindFile, Depth: 2},
			expected: "    deep",
		},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := formatEntryLine(&tc.entry)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatEntryName(t *testing.T) {
	// Test normal path
	normal := "path/to/file"
	assert.Equal(t, "path/to/file", formatEntryName(filepath.FromSlash(normal)))

	// Test long path truncation
	long := make([]byte, MaxEntryLength+100)
	for i := range long {
		long[i] = 'x'
	}
	result := formatEntryName(string(long))
	assert.Equal(t, MaxEntryLength, len(result))
}
