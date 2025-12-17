package tools

import (
	"github.com/anthropics/codex-fork/gosdk/schema"
)

// ReadFileTool returns the read_file tool specification.
// This mirrors create_read_file_tool() in codex-rs/core/src/tools/spec.rs
var ReadFileTool = func() *ToolSpec {
	indentationProperties := map[string]*schema.JSONSchema{
		"anchor_line": schema.Number(
			"Anchor line to center the indentation lookup on (defaults to offset).",
		),
		"max_levels": schema.Number(
			"How many parent indentation levels (smaller indents) to include.",
		),
		"include_siblings": schema.Boolean(
			"When true, include additional blocks that share the anchor indentation.",
		),
		"include_header": schema.Boolean(
			"Include doc comments or attributes directly above the selected block.",
		),
		"max_lines": schema.Number(
			"Hard cap on the number of lines returned when using indentation mode.",
		),
	}

	properties := map[string]*schema.JSONSchema{
		"file_path": schema.String(
			"Absolute path to the file",
		),
		"offset": schema.Number(
			"The line number to start reading from. Must be 1 or greater.",
		),
		"limit": schema.Number(
			"The maximum number of lines to return.",
		),
		"mode": schema.String(
			`Optional mode selector: "slice" for simple ranges (default) or "indentation" to expand around an anchor line.`,
		),
		"indentation": schema.ObjectWithAdditional(
			indentationProperties,
			nil,
			&schema.AdditionalProperties{Allowed: false},
		),
	}

	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        "read_file",
		Description: "Reads a local file with 1-indexed line numbers, supporting slice and indentation-aware block modes.",
		Strict:      false,
		Parameters: schema.Object(
			properties,
			[]string{"file_path"},
		),
	}
}()

// ListDirTool returns the list_dir tool specification.
// This mirrors create_list_dir_tool() in codex-rs/core/src/tools/spec.rs
var ListDirTool = func() *ToolSpec {
	properties := map[string]*schema.JSONSchema{
		"dir_path": schema.String(
			"Absolute path to the directory to list.",
		),
		"offset": schema.Number(
			"The entry number to start listing from. Must be 1 or greater.",
		),
		"limit": schema.Number(
			"The maximum number of entries to return.",
		),
		"depth": schema.Number(
			"The maximum directory depth to traverse. Must be 1 or greater.",
		),
	}

	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        "list_dir",
		Description: "Lists entries in a local directory with 1-indexed entry numbers and simple type labels.",
		Strict:      false,
		Parameters: schema.Object(
			properties,
			[]string{"dir_path"},
		),
	}
}()

// GrepFilesTool returns the grep_files tool specification.
// This mirrors create_grep_files_tool() in codex-rs/core/src/tools/spec.rs
var GrepFilesTool = func() *ToolSpec {
	properties := map[string]*schema.JSONSchema{
		"pattern": schema.String(
			"Regular expression pattern to search for.",
		),
		"include": schema.String(
			`Optional glob that limits which files are searched (e.g. "*.rs" or "*.{ts,tsx}").`,
		),
		"path": schema.String(
			"Directory or file path to search. Defaults to the session's working directory.",
		),
		"limit": schema.Number(
			"Maximum number of file paths to return (defaults to 100).",
		),
	}

	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        "grep_files",
		Description: "Finds files whose contents match the pattern and lists them by modification time.",
		Strict:      false,
		Parameters: schema.Object(
			properties,
			[]string{"pattern"},
		),
	}
}()

// ViewImageTool returns the view_image tool specification.
// This mirrors create_view_image_tool() in codex-rs/core/src/tools/spec.rs
var ViewImageTool = func() *ToolSpec {
	properties := map[string]*schema.JSONSchema{
		"path": schema.String(
			"Local filesystem path to an image file",
		),
	}

	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        "view_image",
		Description: "Attach a local image (by filesystem path) to the conversation context for this turn.",
		Strict:      false,
		Parameters: schema.Object(
			properties,
			[]string{"path"},
		),
	}
}()
