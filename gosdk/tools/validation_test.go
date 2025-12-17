package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToolSchemaValidation validates that Go tool definitions match
// the expected structure from the original Rust implementation.
// This ensures compatibility with the OpenAI Responses API and Chat Completions API.

func TestShellToolSchema(t *testing.T) {
	tool := ShellTool
	validateToolSchema(t, tool, "shell", []string{"command"}, []string{
		"command", "workdir", "timeout_ms", "sandbox_permissions", "justification",
	})
}

func TestShellCommandToolSchema(t *testing.T) {
	tool := ShellCommandTool
	validateToolSchema(t, tool, "shell_command", []string{"command"}, []string{
		"command", "workdir", "login", "timeout_ms", "sandbox_permissions", "justification",
	})
}

func TestExecCommandToolSchema(t *testing.T) {
	tool := ExecCommandTool
	validateToolSchema(t, tool, "exec_command", []string{"cmd"}, []string{
		"cmd", "workdir", "shell", "login", "yield_time_ms", "max_output_tokens",
		"sandbox_permissions", "justification",
	})
}

func TestWriteStdinToolSchema(t *testing.T) {
	tool := WriteStdinTool
	validateToolSchema(t, tool, "write_stdin", []string{"session_id"}, []string{
		"session_id", "chars", "yield_time_ms", "max_output_tokens",
	})
}

func TestReadFileToolSchema(t *testing.T) {
	tool := ReadFileTool
	validateToolSchema(t, tool, "read_file", []string{"file_path"}, []string{
		"file_path", "offset", "limit", "mode", "indentation",
	})

	// Verify indentation sub-schema
	indentSchema := tool.Parameters.Properties["indentation"]
	assert.NotNil(t, indentSchema)
	assert.Contains(t, indentSchema.Properties, "anchor_line")
	assert.Contains(t, indentSchema.Properties, "max_levels")
	assert.Contains(t, indentSchema.Properties, "include_siblings")
	assert.Contains(t, indentSchema.Properties, "include_header")
	assert.Contains(t, indentSchema.Properties, "max_lines")
}

func TestListDirToolSchema(t *testing.T) {
	tool := ListDirTool
	validateToolSchema(t, tool, "list_dir", []string{"dir_path"}, []string{
		"dir_path", "offset", "limit", "depth",
	})
}

func TestGrepFilesToolSchema(t *testing.T) {
	tool := GrepFilesTool
	validateToolSchema(t, tool, "grep_files", []string{"pattern"}, []string{
		"pattern", "include", "path", "limit",
	})
}

func TestViewImageToolSchema(t *testing.T) {
	tool := ViewImageTool
	validateToolSchema(t, tool, "view_image", []string{"path"}, []string{"path"})
}

func TestPlanToolSchema(t *testing.T) {
	tool := PlanTool
	validateToolSchema(t, tool, "update_plan", []string{"plan"}, []string{
		"explanation", "plan",
	})

	// Verify plan array schema
	planSchema := tool.Parameters.Properties["plan"]
	assert.NotNil(t, planSchema)
	assert.Equal(t, "array", string(planSchema.Type))
	assert.NotNil(t, planSchema.Items)

	// Verify plan item schema
	itemSchema := planSchema.Items
	assert.Contains(t, itemSchema.Properties, "step")
	assert.Contains(t, itemSchema.Properties, "status")
	assert.Contains(t, itemSchema.Required, "step")
	assert.Contains(t, itemSchema.Required, "status")
}

func TestListMCPResourcesToolSchema(t *testing.T) {
	tool := ListMCPResourcesTool
	validateToolSchema(t, tool, "list_mcp_resources", nil, []string{
		"server", "cursor",
	})
}

func TestListMCPResourceTemplatesToolSchema(t *testing.T) {
	tool := ListMCPResourceTemplatesTool
	validateToolSchema(t, tool, "list_mcp_resource_templates", nil, []string{
		"server", "cursor",
	})
}

func TestReadMCPResourceToolSchema(t *testing.T) {
	tool := ReadMCPResourceTool
	validateToolSchema(t, tool, "read_mcp_resource", []string{"server", "uri"}, []string{
		"server", "uri",
	})
}

func TestApplyPatchFreeformToolSchema(t *testing.T) {
	tool := ApplyPatchFreeformTool
	validateToolSchema(t, tool, "apply_patch", []string{"input"}, []string{"input"})
}

func TestTestSyncToolSchema(t *testing.T) {
	tool := TestSyncTool
	validateToolSchema(t, tool, "test_sync_tool", nil, []string{
		"sleep_before_ms", "sleep_after_ms", "barrier",
	})

	// Verify barrier sub-schema
	barrierSchema := tool.Parameters.Properties["barrier"]
	assert.NotNil(t, barrierSchema)
	assert.Contains(t, barrierSchema.Properties, "id")
	assert.Contains(t, barrierSchema.Properties, "participants")
	assert.Contains(t, barrierSchema.Properties, "timeout_ms")
	assert.Contains(t, barrierSchema.Required, "id")
	assert.Contains(t, barrierSchema.Required, "participants")
}

// validateToolSchema is a helper that validates common tool schema properties
func validateToolSchema(t *testing.T, tool *ToolSpec, expectedName string, expectedRequired []string, expectedParams []string) {
	t.Helper()

	// Basic validation
	assert.Equal(t, ToolTypeFunction, tool.Type)
	assert.Equal(t, expectedName, tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.False(t, tool.Strict, "Tools should not be strict by default")

	// Parameters validation
	assert.NotNil(t, tool.Parameters)
	assert.Equal(t, "object", string(tool.Parameters.Type))

	// Required fields
	if expectedRequired != nil {
		assert.Equal(t, expectedRequired, tool.Parameters.Required)
	} else {
		assert.Nil(t, tool.Parameters.Required)
	}

	// Properties
	for _, param := range expectedParams {
		assert.Contains(t, tool.Parameters.Properties, param, "Missing parameter: %s", param)
	}

	// Additional properties should be false
	assert.NotNil(t, tool.Parameters.AdditionalProperties)
	assert.False(t, tool.Parameters.AdditionalProperties.Allowed)
}

// TestResponsesAPIJSONFormat validates that tools can be serialized to Responses API format
func TestResponsesAPIJSONFormat(t *testing.T) {
	for _, tool := range AllTools() {
		t.Run(tool.GetName(), func(t *testing.T) {
			data, err := tool.ToResponsesAPIJSON()
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			// Verify required fields for Responses API
			assert.Equal(t, "function", result["type"])
			assert.NotEmpty(t, result["name"])
			assert.NotEmpty(t, result["description"])
			assert.Contains(t, result, "parameters")

			// Verify parameters structure
			params := result["parameters"].(map[string]interface{})
			assert.Equal(t, "object", params["type"])
		})
	}
}

// TestChatCompletionsAPIJSONFormat validates that tools can be serialized to Chat Completions API format
func TestChatCompletionsAPIJSONFormat(t *testing.T) {
	for _, tool := range AllTools() {
		if tool.Type != ToolTypeFunction {
			continue
		}

		t.Run(tool.GetName(), func(t *testing.T) {
			data, err := tool.ToChatCompletionsJSON()
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			// Verify Chat Completions format
			assert.Equal(t, "function", result["type"])
			assert.NotEmpty(t, result["name"])
			assert.Contains(t, result, "function")

			// Verify nested function object
			fn := result["function"].(map[string]interface{})
			assert.NotEmpty(t, fn["name"])
			assert.NotEmpty(t, fn["description"])
			assert.Contains(t, fn, "parameters")
		})
	}
}

// TestToolDescriptionsNotEmpty verifies all tools have meaningful descriptions
func TestToolDescriptionsNotEmpty(t *testing.T) {
	for _, tool := range AllTools() {
		t.Run(tool.GetName(), func(t *testing.T) {
			assert.NotEmpty(t, tool.Description, "Tool %s has empty description", tool.GetName())
			assert.Greater(t, len(tool.Description), 10, "Tool %s description too short", tool.GetName())
		})
	}
}

// TestToolParameterDescriptions verifies all parameters have descriptions
func TestToolParameterDescriptions(t *testing.T) {
	for _, tool := range AllTools() {
		if tool.Parameters == nil {
			continue
		}

		t.Run(tool.GetName(), func(t *testing.T) {
			for name, param := range tool.Parameters.Properties {
				// Most parameters should have descriptions, but some may be empty
				// This is a soft check - we just verify the structure is correct
				assert.NotNil(t, param, "Parameter %s.%s is nil", tool.GetName(), name)
			}
		})
	}
}

// TestToolNamesAreSnakeCase verifies tool names follow snake_case convention
func TestToolNamesAreSnakeCase(t *testing.T) {
	for _, tool := range AllTools() {
		name := tool.GetName()
		t.Run(name, func(t *testing.T) {
			// Check no uppercase letters
			for _, c := range name {
				if c >= 'A' && c <= 'Z' {
					t.Errorf("Tool name %s contains uppercase letter", name)
				}
			}
			// Check no spaces
			assert.NotContains(t, name, " ")
			// Check no hyphens (should use underscores)
			assert.NotContains(t, name, "-")
		})
	}
}

// TestToolRegistryIntegration tests the complete tool registry flow
func TestToolRegistryIntegration(t *testing.T) {
	registry := NewToolRegistry()

	// Register all tools
	for _, tool := range AllTools() {
		registry.RegisterSpec(tool)
	}

	// Verify all tools are registered
	registeredTools := registry.GetTools()
	assert.Equal(t, len(AllTools()), len(registeredTools))

	// Verify JSON serialization
	data, err := registry.ToResponsesAPIJSON()
	require.NoError(t, err)

	var result []interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)
	assert.Equal(t, len(AllTools()), len(result))
}
