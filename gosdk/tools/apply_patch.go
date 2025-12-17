package tools

import (
	"github.com/anthropics/codex-fork/gosdk/schema"
)

// ApplyPatchToolType represents the type of apply_patch tool
type ApplyPatchToolType string

const (
	ApplyPatchFreeform ApplyPatchToolType = "freeform"
	ApplyPatchFunction ApplyPatchToolType = "function"
)

// ApplyPatchFreeformTool returns the apply_patch tool specification in freeform mode.
// This mirrors create_apply_patch_freeform_tool() in codex-rs/core/src/tools/handlers/apply_patch.rs
var ApplyPatchFreeformTool = func() *ToolSpec {
	properties := map[string]*schema.JSONSchema{
		"input": schema.String(
			"The patch to apply in unified diff format",
		),
	}

	return &ToolSpec{
		Type: ToolTypeFunction,
		Name: "apply_patch",
		Description: "Apply a patch to files.\n\n" +
			"Send a single patch per message. Multiple patches should be sent as separate tool calls. " +
			"Never put multiple patches in a single input. After calling apply_patch, the file will be " +
			"automatically rewritten with the new content. You must use apply_patch to write changes to files.\n\n" +
			"Patch format:\n" +
			"*** [ACTION] File: path/to/file\n" +
			"*** NOTE: any notes (optional)\n" +
			"```\n" +
			"[search content]\n" +
			"```\n" +
			"---\n" +
			"```\n" +
			"[replace content]\n" +
			"```\n\n" +
			"Actions:\n" +
			"  - [Add] File: Create new file\n" +
			"  - [Update] File: Update existing file\n" +
			"  - [Delete] File: Delete file\n\n" +
			"For [Update], the search content should match the file content exactly. " +
			"Enclose the search and replace content in triple backticks. " +
			"For [Delete], replace content is optional and can be omitted.",
		Strict: false,
		Parameters: schema.Object(
			properties,
			[]string{"input"},
		),
	}
}()

// ApplyPatchJSONTool returns the apply_patch tool specification in JSON mode.
// This mirrors create_apply_patch_json_tool() in codex-rs/core/src/tools/handlers/apply_patch.rs
var ApplyPatchJSONTool = func() *ToolSpec {
	properties := map[string]*schema.JSONSchema{
		"file_path": schema.String(
			"The path to the file to modify",
		),
		"action": schema.String(
			`The action to perform: "add", "update", or "delete"`,
		),
		"search": schema.String(
			"The content to search for (for update action)",
		),
		"replace": schema.String(
			"The content to replace with",
		),
		"note": schema.String(
			"Optional note about the change",
		),
	}

	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        "apply_patch",
		Description: "Apply a patch to modify a file. Use action 'add' to create new files, 'update' to modify existing files, or 'delete' to remove files.",
		Strict:      false,
		Parameters: schema.Object(
			properties,
			[]string{"file_path", "action"},
		),
	}
}()

// ApplyPatchToolArgs represents the arguments for the apply_patch tool (JSON mode)
type ApplyPatchToolArgs struct {
	FilePath string `json:"file_path"`
	Action   string `json:"action"` // add, update, delete
	Search   string `json:"search,omitempty"`
	Replace  string `json:"replace,omitempty"`
	Note     string `json:"note,omitempty"`
}

// ApplyPatchFreeformArgs represents the arguments for the apply_patch tool (freeform mode)
type ApplyPatchFreeformArgs struct {
	Input string `json:"input"`
}
