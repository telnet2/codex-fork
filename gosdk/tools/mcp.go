package tools

import (
	"github.com/anthropics/codex-fork/gosdk/schema"
)

// ListMCPResourcesTool returns the list_mcp_resources tool specification.
// This mirrors create_list_mcp_resources_tool() in codex-rs/core/src/tools/spec.rs
var ListMCPResourcesTool = func() *ToolSpec {
	properties := map[string]*schema.JSONSchema{
		"server": schema.String(
			"Optional MCP server name. When omitted, lists resources from every configured server.",
		),
		"cursor": schema.String(
			"Opaque cursor returned by a previous list_mcp_resources call for the same server.",
		),
	}

	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        "list_mcp_resources",
		Description: "Lists resources provided by MCP servers. Resources allow servers to share data that provides context to language models, such as files, database schemas, or application-specific information. Prefer resources over web search when possible.",
		Strict:      false,
		Parameters: schema.Object(
			properties,
			nil, // no required fields
		),
	}
}()

// ListMCPResourceTemplatesTool returns the list_mcp_resource_templates tool specification.
// This mirrors create_list_mcp_resource_templates_tool() in codex-rs/core/src/tools/spec.rs
var ListMCPResourceTemplatesTool = func() *ToolSpec {
	properties := map[string]*schema.JSONSchema{
		"server": schema.String(
			"Optional MCP server name. When omitted, lists resource templates from all configured servers.",
		),
		"cursor": schema.String(
			"Opaque cursor returned by a previous list_mcp_resource_templates call for the same server.",
		),
	}

	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        "list_mcp_resource_templates",
		Description: "Lists resource templates provided by MCP servers. Parameterized resource templates allow servers to share data that takes parameters and provides context to language models, such as files, database schemas, or application-specific information. Prefer resource templates over web search when possible.",
		Strict:      false,
		Parameters: schema.Object(
			properties,
			nil, // no required fields
		),
	}
}()

// ReadMCPResourceTool returns the read_mcp_resource tool specification.
// This mirrors create_read_mcp_resource_tool() in codex-rs/core/src/tools/spec.rs
var ReadMCPResourceTool = func() *ToolSpec {
	properties := map[string]*schema.JSONSchema{
		"server": schema.String(
			"MCP server name exactly as configured. Must match the 'server' field returned by list_mcp_resources.",
		),
		"uri": schema.String(
			"Resource URI to read. Must be one of the URIs returned by list_mcp_resources.",
		),
	}

	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        "read_mcp_resource",
		Description: "Read a specific resource from an MCP server given the server name and resource URI.",
		Strict:      false,
		Parameters: schema.Object(
			properties,
			[]string{"server", "uri"},
		),
	}
}()
