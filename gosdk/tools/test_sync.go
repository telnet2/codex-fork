package tools

import (
	"github.com/anthropics/codex-fork/gosdk/schema"
)

// TestSyncTool returns the test_sync_tool specification.
// This mirrors create_test_sync_tool() in codex-rs/core/src/tools/spec.rs
// This tool is used for integration testing synchronization.
var TestSyncTool = func() *ToolSpec {
	barrierProperties := map[string]*schema.JSONSchema{
		"id": schema.String(
			"Identifier shared by concurrent calls that should rendezvous",
		),
		"participants": schema.Number(
			"Number of tool calls that must arrive before the barrier opens",
		),
		"timeout_ms": schema.Number(
			"Maximum time in milliseconds to wait at the barrier",
		),
	}

	properties := map[string]*schema.JSONSchema{
		"sleep_before_ms": schema.Number(
			"Optional delay in milliseconds before any other action",
		),
		"sleep_after_ms": schema.Number(
			"Optional delay in milliseconds after completing the barrier",
		),
		"barrier": schema.ObjectWithAdditional(
			barrierProperties,
			[]string{"id", "participants"},
			&schema.AdditionalProperties{Allowed: false},
		),
	}

	return &ToolSpec{
		Type:        ToolTypeFunction,
		Name:        "test_sync_tool",
		Description: "Internal synchronization helper used by Codex integration tests.",
		Strict:      false,
		Parameters: schema.Object(
			properties,
			nil, // no required fields
		),
	}
}()

// TestSyncToolArgs represents the arguments for the test_sync_tool
type TestSyncToolArgs struct {
	SleepBeforeMs *int64                `json:"sleep_before_ms,omitempty"`
	SleepAfterMs  *int64                `json:"sleep_after_ms,omitempty"`
	Barrier       *TestSyncBarrierArgs  `json:"barrier,omitempty"`
}

// TestSyncBarrierArgs represents the barrier configuration
type TestSyncBarrierArgs struct {
	ID           string `json:"id"`
	Participants int    `json:"participants"`
	TimeoutMs    *int64 `json:"timeout_ms,omitempty"`
}
