package tools

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShellToolDefinition(t *testing.T) {
	tool := ShellTool

	assert.Equal(t, ToolTypeFunction, tool.Type)
	assert.Equal(t, "shell", tool.Name)
	assert.False(t, tool.Strict)
	assert.NotEmpty(t, tool.Description)

	// Verify parameters
	assert.NotNil(t, tool.Parameters)
	assert.Equal(t, "command", tool.Parameters.Required[0])

	// Verify command parameter is an array
	cmdParam := tool.Parameters.Properties["command"]
	assert.NotNil(t, cmdParam)
	assert.Equal(t, "array", string(cmdParam.Type))
}

func TestShellCommandToolDefinition(t *testing.T) {
	tool := ShellCommandTool

	assert.Equal(t, ToolTypeFunction, tool.Type)
	assert.Equal(t, "shell_command", tool.Name)
	assert.False(t, tool.Strict)

	// Verify parameters
	cmdParam := tool.Parameters.Properties["command"]
	assert.NotNil(t, cmdParam)
	assert.Equal(t, "string", string(cmdParam.Type))
}

func TestShellToolDescriptionByOS(t *testing.T) {
	tool := ShellTool

	if runtime.GOOS == "windows" {
		assert.Contains(t, tool.Description, "Powershell")
		assert.Contains(t, tool.Description, "CreateProcessW")
	} else {
		assert.Contains(t, tool.Description, "execvp")
		assert.Contains(t, tool.Description, "bash")
	}
}

func TestExecCommandToolDefinition(t *testing.T) {
	tool := ExecCommandTool

	assert.Equal(t, ToolTypeFunction, tool.Type)
	assert.Equal(t, "exec_command", tool.Name)
	assert.Contains(t, tool.Description, "PTY")

	// Verify required parameters
	assert.Contains(t, tool.Parameters.Required, "cmd")

	// Verify all expected parameters exist
	expectedParams := []string{"cmd", "workdir", "shell", "login", "yield_time_ms", "max_output_tokens", "sandbox_permissions", "justification"}
	for _, param := range expectedParams {
		assert.Contains(t, tool.Parameters.Properties, param, "Missing parameter: %s", param)
	}
}

func TestWriteStdinToolDefinition(t *testing.T) {
	tool := WriteStdinTool

	assert.Equal(t, ToolTypeFunction, tool.Type)
	assert.Equal(t, "write_stdin", tool.Name)
	assert.Contains(t, tool.Description, "unified exec session")

	// Verify required parameters
	assert.Contains(t, tool.Parameters.Required, "session_id")
}

func TestReadFileToolDefinition(t *testing.T) {
	tool := ReadFileTool

	assert.Equal(t, ToolTypeFunction, tool.Type)
	assert.Equal(t, "read_file", tool.Name)
	assert.Contains(t, tool.Description, "1-indexed line numbers")

	// Verify required parameters
	assert.Contains(t, tool.Parameters.Required, "file_path")

	// Verify indentation parameter exists
	indentParam := tool.Parameters.Properties["indentation"]
	assert.NotNil(t, indentParam)
	assert.Equal(t, "object", string(indentParam.Type))
}

func TestListDirToolDefinition(t *testing.T) {
	tool := ListDirTool

	assert.Equal(t, ToolTypeFunction, tool.Type)
	assert.Equal(t, "list_dir", tool.Name)
	assert.Contains(t, tool.Description, "1-indexed entry numbers")

	// Verify required parameters
	assert.Contains(t, tool.Parameters.Required, "dir_path")

	// Verify expected parameters
	expectedParams := []string{"dir_path", "offset", "limit", "depth"}
	for _, param := range expectedParams {
		assert.Contains(t, tool.Parameters.Properties, param)
	}
}

func TestGrepFilesToolDefinition(t *testing.T) {
	tool := GrepFilesTool

	assert.Equal(t, ToolTypeFunction, tool.Type)
	assert.Equal(t, "grep_files", tool.Name)
	assert.Contains(t, tool.Description, "pattern")

	// Verify required parameters
	assert.Contains(t, tool.Parameters.Required, "pattern")

	// Verify include parameter description
	includeParam := tool.Parameters.Properties["include"]
	assert.NotNil(t, includeParam)
	assert.Contains(t, includeParam.Description, "glob")
}

func TestViewImageToolDefinition(t *testing.T) {
	tool := ViewImageTool

	assert.Equal(t, ToolTypeFunction, tool.Type)
	assert.Equal(t, "view_image", tool.Name)
	assert.Contains(t, tool.Description, "image")

	// Verify required parameters
	assert.Contains(t, tool.Parameters.Required, "path")
}

func TestPlanToolDefinition(t *testing.T) {
	tool := PlanTool

	assert.Equal(t, ToolTypeFunction, tool.Type)
	assert.Equal(t, "update_plan", tool.Name)
	assert.Contains(t, tool.Description, "task plan")

	// Verify required parameters
	assert.Contains(t, tool.Parameters.Required, "plan")

	// Verify plan parameter is an array
	planParam := tool.Parameters.Properties["plan"]
	assert.NotNil(t, planParam)
	assert.Equal(t, "array", string(planParam.Type))

	// Verify plan item schema
	assert.NotNil(t, planParam.Items)
	assert.Contains(t, planParam.Items.Properties, "step")
	assert.Contains(t, planParam.Items.Properties, "status")
}

func TestMCPToolsDefinitions(t *testing.T) {
	tests := []struct {
		tool        *ToolSpec
		name        string
		description string
		required    []string
	}{
		{
			tool:        ListMCPResourcesTool,
			name:        "list_mcp_resources",
			description: "Lists resources",
			required:    nil,
		},
		{
			tool:        ListMCPResourceTemplatesTool,
			name:        "list_mcp_resource_templates",
			description: "resource templates",
			required:    nil,
		},
		{
			tool:        ReadMCPResourceTool,
			name:        "read_mcp_resource",
			description: "Read a specific resource",
			required:    []string{"server", "uri"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, ToolTypeFunction, tc.tool.Type)
			assert.Equal(t, tc.name, tc.tool.Name)
			assert.Contains(t, tc.tool.Description, tc.description)
			if tc.required != nil {
				assert.Equal(t, tc.required, tc.tool.Parameters.Required)
			}
		})
	}
}

func TestApplyPatchToolDefinition(t *testing.T) {
	tool := ApplyPatchFreeformTool

	assert.Equal(t, ToolTypeFunction, tool.Type)
	assert.Equal(t, "apply_patch", tool.Name)
	assert.Contains(t, tool.Description, "patch")

	// Verify required parameters
	assert.Contains(t, tool.Parameters.Required, "input")
}

func TestTestSyncToolDefinition(t *testing.T) {
	tool := TestSyncTool

	assert.Equal(t, ToolTypeFunction, tool.Type)
	assert.Equal(t, "test_sync_tool", tool.Name)
	assert.Contains(t, tool.Description, "integration tests")

	// Verify barrier parameter
	barrierParam := tool.Parameters.Properties["barrier"]
	assert.NotNil(t, barrierParam)
	assert.Contains(t, barrierParam.Properties, "id")
	assert.Contains(t, barrierParam.Properties, "participants")
}

func TestAllTools(t *testing.T) {
	tools := AllTools()
	assert.Greater(t, len(tools), 0)

	// Verify no duplicates
	names := make(map[string]bool)
	for _, tool := range tools {
		name := tool.GetName()
		assert.False(t, names[name], "Duplicate tool name: %s", name)
		names[name] = true
	}
}

func TestDefaultTools(t *testing.T) {
	tools := DefaultTools()
	assert.Greater(t, len(tools), 0)

	// Verify expected default tools
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.GetName()] = true
	}

	expectedDefaults := []string{
		"shell_command",
		"list_mcp_resources",
		"update_plan",
		"apply_patch",
		"view_image",
	}
	for _, name := range expectedDefaults {
		assert.True(t, names[name], "Expected default tool: %s", name)
	}
}

func TestToolsByName(t *testing.T) {
	tools := ToolsByName()

	// Verify we can look up tools by name
	shellTool := tools["shell"]
	assert.NotNil(t, shellTool)
	assert.Equal(t, "shell", shellTool.Name)

	planTool := tools["update_plan"]
	assert.NotNil(t, planTool)
	assert.Equal(t, "update_plan", planTool.Name)
}

func TestGetToolByName(t *testing.T) {
	tool := GetToolByName("read_file")
	assert.NotNil(t, tool)
	assert.Equal(t, "read_file", tool.Name)

	notFound := GetToolByName("nonexistent")
	assert.Nil(t, notFound)
}

func TestToolToResponsesAPIJSON(t *testing.T) {
	tool := ShellTool

	data, err := tool.ToResponsesAPIJSON()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "function", result["type"])
	assert.Equal(t, "shell", result["name"])
	assert.NotEmpty(t, result["description"])
	assert.Contains(t, result, "parameters")
}

func TestToolToChatCompletionsJSON(t *testing.T) {
	tool := ShellTool

	data, err := tool.ToChatCompletionsJSON()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "function", result["type"])
	assert.Equal(t, "shell", result["name"])
	assert.Contains(t, result, "function")

	fn := result["function"].(map[string]interface{})
	assert.Equal(t, "shell", fn["name"])
	assert.NotEmpty(t, fn["description"])
}

func TestToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	// Register a tool
	registry.RegisterSpec(ShellTool)
	registry.RegisterSpec(ReadFileTool)

	tools := registry.GetTools()
	assert.Len(t, tools, 2)
}

func TestLocalShellTool(t *testing.T) {
	tool := LocalShellTool()
	assert.Equal(t, ToolTypeLocalShell, tool.Type)
	assert.Equal(t, "local_shell", tool.GetName())
}

func TestWebSearchTool(t *testing.T) {
	tool := WebSearchTool()
	assert.Equal(t, ToolTypeWebSearch, tool.Type)
	assert.Equal(t, "web_search", tool.GetName())
}

func TestToolOutput(t *testing.T) {
	// Test successful output
	output := NewToolOutput("Success message")
	assert.Equal(t, "Success message", output.Content)
	assert.True(t, *output.Success)

	// Test error output
	errOutput := NewToolOutputError("Error message")
	assert.Equal(t, "Error message", errOutput.Content)
	assert.False(t, *errOutput.Success)
}

func TestExperimentalTools(t *testing.T) {
	tools := ExperimentalTools()
	assert.Greater(t, len(tools), 0)

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.GetName()] = true
	}

	// Verify experimental tools
	assert.True(t, names["read_file"])
	assert.True(t, names["grep_files"])
	assert.True(t, names["list_dir"])
	assert.True(t, names["test_sync_tool"])
}

func TestToolsForUnifiedExec(t *testing.T) {
	tools := ToolsForUnifiedExec()
	assert.Greater(t, len(tools), 0)

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.GetName()] = true
	}

	// Verify unified exec tools
	assert.True(t, names["exec_command"])
	assert.True(t, names["write_stdin"])
}
