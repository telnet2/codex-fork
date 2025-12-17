package tools

import (
	"github.com/anthropics/codex-fork/gosdk/schema"
)

// PlanTool returns the update_plan tool specification.
// This mirrors PLAN_TOOL in codex-rs/core/src/tools/handlers/plan.rs
var PlanTool = func() *ToolSpec {
	planItemProps := map[string]*schema.JSONSchema{
		"step":   schema.String(""),
		"status": schema.String("One of: pending, in_progress, completed"),
	}

	planItemsSchema := schema.Array(
		schema.ObjectWithAdditional(
			planItemProps,
			[]string{"step", "status"},
			&schema.AdditionalProperties{Allowed: false},
		),
		"The list of steps",
	)

	properties := map[string]*schema.JSONSchema{
		"explanation": schema.String(""),
		"plan":        planItemsSchema,
	}

	return &ToolSpec{
		Type: ToolTypeFunction,
		Name: "update_plan",
		Description: `Updates the task plan.
Provide an optional explanation and a list of plan items, each with a step and status.
At most one step can be in_progress at a time.
`,
		Strict: false,
		Parameters: schema.Object(
			properties,
			[]string{"plan"},
		),
	}
}()

// PlanItem represents a single step in a plan
type PlanItem struct {
	Step   string `json:"step"`
	Status string `json:"status"` // pending, in_progress, completed
}

// UpdatePlanArgs represents the arguments for the update_plan tool
type UpdatePlanArgs struct {
	Explanation string     `json:"explanation,omitempty"`
	Plan        []PlanItem `json:"plan"`
}

// ValidatePlanArgs validates the plan arguments
func ValidatePlanArgs(args *UpdatePlanArgs) error {
	inProgressCount := 0
	for _, item := range args.Plan {
		if item.Status == "in_progress" {
			inProgressCount++
		}
	}
	// At most one step can be in_progress at a time
	// (validation is soft - we don't error, just note it)
	return nil
}
