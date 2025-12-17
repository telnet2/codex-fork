// Package tools provides tool definitions for the Codex assistant.
// This mirrors the Rust implementation in codex-rs/core/src/tools/spec.rs
package tools

import (
	"encoding/json"

	"github.com/anthropics/codex-fork/gosdk/schema"
)

// ToolType represents the type of tool
type ToolType string

const (
	ToolTypeFunction   ToolType = "function"
	ToolTypeLocalShell ToolType = "local_shell"
	ToolTypeWebSearch  ToolType = "web_search"
	ToolTypeFreeform   ToolType = "freeform"
)

// ToolSpec represents a tool specification that can be sent to the API.
// This mirrors the Rust ToolSpec enum.
type ToolSpec struct {
	Type        ToolType       `json:"type"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Strict      bool           `json:"strict,omitempty"`
	Parameters  *schema.JSONSchema `json:"parameters,omitempty"`
}

// ResponsesAPITool represents a function tool compatible with the OpenAI Responses API.
// This mirrors the Rust ResponsesApiTool struct.
type ResponsesAPITool struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Strict      bool               `json:"strict"`
	Parameters  *schema.JSONSchema `json:"parameters"`
}

// ToToolSpec converts a ResponsesAPITool to a ToolSpec
func (r *ResponsesAPITool) ToToolSpec() *ToolSpec {
	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        r.Name,
		Description: r.Description,
		Strict:      r.Strict,
		Parameters:  r.Parameters,
	}
}

// FunctionTool creates a new function tool spec
func FunctionTool(name, description string, parameters *schema.JSONSchema) *ToolSpec {
	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        name,
		Description: description,
		Strict:      false,
		Parameters:  parameters,
	}
}

// LocalShellTool creates a local shell tool spec
func LocalShellTool() *ToolSpec {
	return &ToolSpec{
		Type: ToolTypeLocalShell,
	}
}

// WebSearchTool creates a web search tool spec
func WebSearchTool() *ToolSpec {
	return &ToolSpec{
		Type: ToolTypeWebSearch,
	}
}

// GetName returns the tool name
func (t *ToolSpec) GetName() string {
	switch t.Type {
	case ToolTypeLocalShell:
		return "local_shell"
	case ToolTypeWebSearch:
		return "web_search"
	default:
		return t.Name
	}
}

// ToResponsesAPIJSON converts the tool spec to JSON compatible with the Responses API
func (t *ToolSpec) ToResponsesAPIJSON() ([]byte, error) {
	return json.Marshal(t)
}

// ToChatCompletionsJSON converts the tool spec to JSON compatible with the Chat Completions API
func (t *ToolSpec) ToChatCompletionsJSON() ([]byte, error) {
	if t.Type != ToolTypeFunction {
		return nil, nil
	}

	chatTool := map[string]interface{}{
		"type": "function",
		"name": t.Name,
		"function": map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"strict":      t.Strict,
			"parameters":  t.Parameters,
		},
	}

	return json.Marshal(chatTool)
}

// ToolRegistry holds a collection of tool specs
type ToolRegistry struct {
	tools    []*ToolSpec
	handlers map[string]ToolHandler
}

// ToolHandler is the interface that tool handlers must implement
type ToolHandler interface {
	// Name returns the tool name
	Name() string
	// Handle executes the tool with the given arguments
	Handle(arguments string) (*ToolOutput, error)
}

// ToolOutput represents the output from a tool execution
type ToolOutput struct {
	Content      string                 `json:"content"`
	ContentItems []map[string]interface{} `json:"content_items,omitempty"`
	Success      *bool                  `json:"success,omitempty"`
}

// NewToolOutput creates a successful tool output
func NewToolOutput(content string) *ToolOutput {
	success := true
	return &ToolOutput{
		Content: content,
		Success: &success,
	}
}

// NewToolOutputError creates an error tool output
func NewToolOutputError(message string) *ToolOutput {
	success := false
	return &ToolOutput{
		Content: message,
		Success: &success,
	}
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:    make([]*ToolSpec, 0),
		handlers: make(map[string]ToolHandler),
	}
}

// Register adds a tool spec and its handler to the registry
func (r *ToolRegistry) Register(spec *ToolSpec, handler ToolHandler) {
	r.tools = append(r.tools, spec)
	if handler != nil {
		r.handlers[spec.GetName()] = handler
	}
}

// RegisterSpec adds a tool spec without a handler
func (r *ToolRegistry) RegisterSpec(spec *ToolSpec) {
	r.tools = append(r.tools, spec)
}

// GetTools returns all registered tool specs
func (r *ToolRegistry) GetTools() []*ToolSpec {
	return r.tools
}

// GetHandler returns the handler for a given tool name
func (r *ToolRegistry) GetHandler(name string) (ToolHandler, bool) {
	handler, ok := r.handlers[name]
	return handler, ok
}

// ToResponsesAPIJSON converts all tools to JSON for the Responses API
func (r *ToolRegistry) ToResponsesAPIJSON() ([]byte, error) {
	return json.Marshal(r.tools)
}
