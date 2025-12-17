// Package handlers provides tool handler implementations.
package handlers

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/codex-fork/gosdk/tools"
)

const (
	// MaxLineLength is the maximum length of a line before truncation
	MaxLineLength = 500
	// TabWidth is the number of spaces per tab for indentation calculation
	TabWidth = 4
)

// ReadMode represents the mode for reading files
type ReadMode string

const (
	ReadModeSlice       ReadMode = "slice"
	ReadModeIndentation ReadMode = "indentation"
)

// ReadFileArgs represents the arguments for the read_file tool
type ReadFileArgs struct {
	FilePath    string           `json:"file_path"`
	Offset      int              `json:"offset"`
	Limit       int              `json:"limit"`
	Mode        ReadMode         `json:"mode"`
	Indentation *IndentationArgs `json:"indentation,omitempty"`
}

// IndentationArgs represents indentation-specific arguments
type IndentationArgs struct {
	AnchorLine      *int  `json:"anchor_line,omitempty"`
	MaxLevels       int   `json:"max_levels"`
	IncludeSiblings bool  `json:"include_siblings"`
	IncludeHeader   bool  `json:"include_header"`
	MaxLines        *int  `json:"max_lines,omitempty"`
}

// ReadFileHandler implements the read_file tool
type ReadFileHandler struct{}

// NewReadFileHandler creates a new ReadFileHandler
func NewReadFileHandler() *ReadFileHandler {
	return &ReadFileHandler{}
}

// Name returns the tool name
func (h *ReadFileHandler) Name() string {
	return "read_file"
}

// Handle executes the read_file tool
func (h *ReadFileHandler) Handle(arguments string) (*tools.ToolOutput, error) {
	var args ReadFileArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse function arguments: %w", err)
	}

	// Set defaults
	if args.Offset == 0 {
		args.Offset = 1
	}
	if args.Limit == 0 {
		args.Limit = 2000
	}
	if args.Mode == "" {
		args.Mode = ReadModeSlice
	}

	// Validate arguments
	if args.Offset < 1 {
		return tools.NewToolOutputError("offset must be a 1-indexed line number"), nil
	}
	if args.Limit < 1 {
		return tools.NewToolOutputError("limit must be greater than zero"), nil
	}
	if !filepath.IsAbs(args.FilePath) {
		return tools.NewToolOutputError("file_path must be an absolute path"), nil
	}

	var lines []string
	var err error

	switch args.Mode {
	case ReadModeSlice:
		lines, err = readSlice(args.FilePath, args.Offset, args.Limit)
	case ReadModeIndentation:
		indentArgs := args.Indentation
		if indentArgs == nil {
			indentArgs = &IndentationArgs{
				MaxLevels:       0,
				IncludeSiblings: false,
				IncludeHeader:   true,
			}
		}
		lines, err = readIndentation(args.FilePath, args.Offset, args.Limit, indentArgs)
	default:
		return tools.NewToolOutputError(fmt.Sprintf("unknown mode: %s", args.Mode)), nil
	}

	if err != nil {
		return tools.NewToolOutputError(err.Error()), nil
	}

	return tools.NewToolOutput(strings.Join(lines, "\n")), nil
}

// readSlice reads a simple slice of lines from a file
func readSlice(filePath string, offset, limit int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max line

	var lines []string
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if lineNum < offset {
			continue
		}
		if len(lines) >= limit {
			break
		}

		line := formatLine(scanner.Text())
		lines = append(lines, fmt.Sprintf("L%d: %s", lineNum, line))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if lineNum < offset {
		return nil, errors.New("offset exceeds file length")
	}

	return lines, nil
}

// readIndentation reads lines with indentation-aware block expansion
func readIndentation(filePath string, offset, limit int, opts *IndentationArgs) ([]string, error) {
	anchorLine := offset
	if opts.AnchorLine != nil {
		anchorLine = *opts.AnchorLine
	}
	if anchorLine < 1 {
		return nil, errors.New("anchor_line must be a 1-indexed line number")
	}

	guardLimit := limit
	if opts.MaxLines != nil {
		guardLimit = *opts.MaxLines
	}
	if guardLimit < 1 {
		return nil, errors.New("max_lines must be greater than zero")
	}

	// Read all lines
	allLines, err := readAllLines(filePath)
	if err != nil {
		return nil, err
	}

	if len(allLines) == 0 || anchorLine > len(allLines) {
		return nil, errors.New("anchor_line exceeds file length")
	}

	anchorIndex := anchorLine - 1
	effectiveIndents := computeEffectiveIndents(allLines)
	anchorIndent := effectiveIndents[anchorIndex]

	// Compute min indent
	minIndent := 0
	if opts.MaxLevels > 0 {
		minIndent = anchorIndent - opts.MaxLevels*TabWidth
		if minIndent < 0 {
			minIndent = 0
		}
	}

	// Cap requested lines
	finalLimit := min(limit, guardLimit)
	if finalLimit > len(allLines) {
		finalLimit = len(allLines)
	}

	if finalLimit == 1 {
		return []string{fmt.Sprintf("L%d: %s", allLines[anchorIndex].Number, allLines[anchorIndex].Display)}, nil
	}

	// Build output using bidirectional expansion
	var result []*lineRecord
	result = append(result, allLines[anchorIndex])

	i := anchorIndex - 1 // up cursor
	j := anchorIndex + 1 // down cursor
	iCounterMinIndent := 0
	jCounterMinIndent := 0

	for len(result) < finalLimit {
		progressed := 0

		// Expand up
		if i >= 0 {
			if effectiveIndents[i] >= minIndent {
				result = append([]*lineRecord{allLines[i]}, result...)
				progressed++
				i--

				// Handle siblings
				if i >= 0 && effectiveIndents[i+1] == minIndent && !opts.IncludeSiblings {
					allowHeaderComment := opts.IncludeHeader && allLines[i+1].IsComment()
					canTakeLine := allowHeaderComment || iCounterMinIndent == 0

					if canTakeLine {
						iCounterMinIndent++
					} else {
						result = result[1:]
						progressed--
						i = -1
					}
				}

				if len(result) >= finalLimit {
					break
				}
			} else {
				i = -1
			}
		}

		// Expand down
		if j < len(allLines) {
			if effectiveIndents[j] >= minIndent {
				result = append(result, allLines[j])
				progressed++
				j++

				// Handle siblings
				if effectiveIndents[j-1] == minIndent && !opts.IncludeSiblings {
					if jCounterMinIndent > 0 {
						result = result[:len(result)-1]
						progressed--
						j = len(allLines)
					}
					jCounterMinIndent++
				}
			} else {
				j = len(allLines)
			}
		}

		if progressed == 0 {
			break
		}
	}

	// Trim empty lines
	result = trimEmptyLines(result)

	// Format output
	lines := make([]string, len(result))
	for idx, record := range result {
		lines[idx] = fmt.Sprintf("L%d: %s", record.Number, record.Display)
	}

	return lines, nil
}

// lineRecord represents a line with metadata
type lineRecord struct {
	Number  int
	Raw     string
	Display string
	Indent  int
}

// IsBlank returns true if the line is blank
func (l *lineRecord) IsBlank() bool {
	return strings.TrimSpace(l.Raw) == ""
}

// IsComment returns true if the line is a comment
func (l *lineRecord) IsComment() bool {
	trimmed := strings.TrimSpace(l.Raw)
	prefixes := []string{"#", "//", "--"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// readAllLines reads all lines from a file into lineRecords
func readAllLines(filePath string) ([]*lineRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var lines []*lineRecord
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		raw := scanner.Text()
		display := formatLine(raw)
		indent := measureIndent(raw)
		lines = append(lines, &lineRecord{
			Number:  lineNum,
			Raw:     raw,
			Display: display,
			Indent:  indent,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return lines, nil
}

// computeEffectiveIndents computes effective indentation for each line
func computeEffectiveIndents(records []*lineRecord) []int {
	effective := make([]int, len(records))
	previousIndent := 0

	for i, record := range records {
		if record.IsBlank() {
			effective[i] = previousIndent
		} else {
			previousIndent = record.Indent
			effective[i] = previousIndent
		}
	}

	return effective
}

// measureIndent measures the indentation of a line
func measureIndent(line string) int {
	indent := 0
	for _, c := range line {
		switch c {
		case ' ':
			indent++
		case '\t':
			indent += TabWidth
		default:
			return indent
		}
	}
	return indent
}

// formatLine formats a line for display, truncating if necessary
func formatLine(line string) string {
	if len(line) > MaxLineLength {
		return line[:MaxLineLength]
	}
	return line
}

// trimEmptyLines removes leading and trailing empty lines
func trimEmptyLines(lines []*lineRecord) []*lineRecord {
	// Trim leading
	for len(lines) > 0 && lines[0].IsBlank() {
		lines = lines[1:]
	}
	// Trim trailing
	for len(lines) > 0 && lines[len(lines)-1].IsBlank() {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
