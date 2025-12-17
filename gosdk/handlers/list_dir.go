package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anthropics/codex-fork/gosdk/tools"
)

const (
	// MaxEntryLength is the maximum length of an entry name before truncation
	MaxEntryLength = 500
	// IndentationSpaces is the number of spaces per depth level
	IndentationSpaces = 2
)

// ListDirArgs represents the arguments for the list_dir tool
type ListDirArgs struct {
	DirPath string `json:"dir_path"`
	Offset  int    `json:"offset"`
	Limit   int    `json:"limit"`
	Depth   int    `json:"depth"`
}

// DirEntryKind represents the type of directory entry
type DirEntryKind int

const (
	KindFile DirEntryKind = iota
	KindDirectory
	KindSymlink
	KindOther
)

// DirEntry represents a directory entry
type DirEntry struct {
	Name        string
	DisplayName string
	Depth       int
	Kind        DirEntryKind
}

// ListDirHandler implements the list_dir tool
type ListDirHandler struct{}

// NewListDirHandler creates a new ListDirHandler
func NewListDirHandler() *ListDirHandler {
	return &ListDirHandler{}
}

// Name returns the tool name
func (h *ListDirHandler) Name() string {
	return "list_dir"
}

// Handle executes the list_dir tool
func (h *ListDirHandler) Handle(arguments string) (*tools.ToolOutput, error) {
	var args ListDirArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse function arguments: %w", err)
	}

	// Set defaults
	if args.Offset == 0 {
		args.Offset = 1
	}
	if args.Limit == 0 {
		args.Limit = 25
	}
	if args.Depth == 0 {
		args.Depth = 2
	}

	// Validate arguments
	if args.Offset < 1 {
		return tools.NewToolOutputError("offset must be a 1-indexed entry number"), nil
	}
	if args.Limit < 1 {
		return tools.NewToolOutputError("limit must be greater than zero"), nil
	}
	if args.Depth < 1 {
		return tools.NewToolOutputError("depth must be greater than zero"), nil
	}
	if !filepath.IsAbs(args.DirPath) {
		return tools.NewToolOutputError("dir_path must be an absolute path"), nil
	}

	entries, err := listDirSlice(args.DirPath, args.Offset, args.Limit, args.Depth)
	if err != nil {
		return tools.NewToolOutputError(err.Error()), nil
	}

	// Build output
	output := []string{fmt.Sprintf("Absolute path: %s", args.DirPath)}
	output = append(output, entries...)

	return tools.NewToolOutput(strings.Join(output, "\n")), nil
}

// listDirSlice lists directory entries with pagination
func listDirSlice(dirPath string, offset, limit, depth int) ([]string, error) {
	var entries []DirEntry
	if err := collectEntries(dirPath, "", depth, &entries); err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, nil
	}

	startIndex := offset - 1
	if startIndex >= len(entries) {
		return nil, fmt.Errorf("offset exceeds directory entry count")
	}

	remainingEntries := len(entries) - startIndex
	cappedLimit := limit
	if cappedLimit > remainingEntries {
		cappedLimit = remainingEntries
	}
	endIndex := startIndex + cappedLimit

	selectedEntries := entries[startIndex:endIndex]
	sort.Slice(selectedEntries, func(i, j int) bool {
		return selectedEntries[i].Name < selectedEntries[j].Name
	})

	formatted := make([]string, 0, len(selectedEntries)+1)
	for _, entry := range selectedEntries {
		formatted = append(formatted, formatEntryLine(&entry))
	}

	if endIndex < len(entries) {
		formatted = append(formatted, fmt.Sprintf("More than %d entries found", cappedLimit))
	}

	return formatted, nil
}

// collectEntries collects directory entries recursively using BFS
func collectEntries(dirPath, relativePrefix string, depth int, entries *[]DirEntry) error {
	type queueItem struct {
		currentDir     string
		prefix         string
		remainingDepth int
	}

	queue := []queueItem{{dirPath, relativePrefix, depth}}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		dir, err := os.ReadDir(item.currentDir)
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}

		type dirEntryInfo struct {
			entryPath    string
			relativePath string
			kind         DirEntryKind
			entry        DirEntry
		}

		var dirEntries []dirEntryInfo
		for _, entry := range dir {
			fileName := entry.Name()
			relativePath := fileName
			if item.prefix != "" {
				relativePath = filepath.Join(item.prefix, fileName)
			}

			displayName := formatEntryComponent(fileName)
			displayDepth := len(strings.Split(item.prefix, string(filepath.Separator)))
			if item.prefix == "" {
				displayDepth = 0
			}
			sortKey := formatEntryName(relativePath)

			info, err := entry.Info()
			if err != nil {
				continue
			}

			kind := getEntryKind(info)
			dirEntries = append(dirEntries, dirEntryInfo{
				entryPath:    filepath.Join(item.currentDir, fileName),
				relativePath: relativePath,
				kind:         kind,
				entry: DirEntry{
					Name:        sortKey,
					DisplayName: displayName,
					Depth:       displayDepth,
					Kind:        kind,
				},
			})
		}

		// Sort entries
		sort.Slice(dirEntries, func(i, j int) bool {
			return dirEntries[i].entry.Name < dirEntries[j].entry.Name
		})

		for _, de := range dirEntries {
			if de.kind == KindDirectory && item.remainingDepth > 1 {
				queue = append(queue, queueItem{
					currentDir:     de.entryPath,
					prefix:         de.relativePath,
					remainingDepth: item.remainingDepth - 1,
				})
			}
			*entries = append(*entries, de.entry)
		}
	}

	return nil
}

// getEntryKind determines the kind of a file
func getEntryKind(info os.FileInfo) DirEntryKind {
	mode := info.Mode()
	if mode&os.ModeSymlink != 0 {
		return KindSymlink
	}
	if mode.IsDir() {
		return KindDirectory
	}
	if mode.IsRegular() {
		return KindFile
	}
	return KindOther
}

// formatEntryName formats a path for sorting
func formatEntryName(path string) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	if len(normalized) > MaxEntryLength {
		return normalized[:MaxEntryLength]
	}
	return normalized
}

// formatEntryComponent formats a single path component
func formatEntryComponent(name string) string {
	if len(name) > MaxEntryLength {
		return name[:MaxEntryLength]
	}
	return name
}

// formatEntryLine formats an entry for display
func formatEntryLine(entry *DirEntry) string {
	indent := strings.Repeat(" ", entry.Depth*IndentationSpaces)
	name := entry.DisplayName

	switch entry.Kind {
	case KindDirectory:
		name += "/"
	case KindSymlink:
		name += "@"
	case KindOther:
		name += "?"
	}

	return indent + name
}
