package tools

// AllTools returns all available tool specifications.
// This provides a convenient way to access all tools defined in the SDK.
func AllTools() []*ToolSpec {
	return []*ToolSpec{
		// Shell tools
		ShellTool,
		ShellCommandTool,
		ExecCommandTool,
		WriteStdinTool,

		// File operation tools
		ReadFileTool,
		ListDirTool,
		GrepFilesTool,
		ViewImageTool,

		// Planning tool
		PlanTool,

		// Apply patch tools
		ApplyPatchFreeformTool,

		// MCP tools
		ListMCPResourcesTool,
		ListMCPResourceTemplatesTool,
		ReadMCPResourceTool,

		// Test tools
		TestSyncTool,
	}
}

// DefaultTools returns the default set of tools for typical usage.
// This excludes experimental and test tools.
func DefaultTools() []*ToolSpec {
	return []*ToolSpec{
		ShellCommandTool,
		ListMCPResourcesTool,
		ListMCPResourceTemplatesTool,
		ReadMCPResourceTool,
		PlanTool,
		ApplyPatchFreeformTool,
		ViewImageTool,
	}
}

// ToolsByName returns a map of tool names to their specifications.
func ToolsByName() map[string]*ToolSpec {
	tools := AllTools()
	result := make(map[string]*ToolSpec, len(tools))
	for _, tool := range tools {
		result[tool.GetName()] = tool
	}
	return result
}

// GetToolByName returns a tool specification by name, or nil if not found.
func GetToolByName(name string) *ToolSpec {
	tools := ToolsByName()
	return tools[name]
}

// ExperimentalTools returns tools that are marked as experimental.
func ExperimentalTools() []*ToolSpec {
	return []*ToolSpec{
		ReadFileTool,
		GrepFilesTool,
		ListDirTool,
		TestSyncTool,
	}
}

// ToolsForUnifiedExec returns the tools used when unified exec is enabled.
func ToolsForUnifiedExec() []*ToolSpec {
	return []*ToolSpec{
		ExecCommandTool,
		WriteStdinTool,
		ListMCPResourcesTool,
		ListMCPResourceTemplatesTool,
		ReadMCPResourceTool,
		PlanTool,
		ApplyPatchFreeformTool,
		ViewImageTool,
	}
}
