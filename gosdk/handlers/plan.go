package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/anthropics/codex-fork/gosdk/tools"
)

// PlanHandler implements the update_plan tool
type PlanHandler struct {
	// OnPlanUpdate is called when a plan is updated
	OnPlanUpdate func(args *tools.UpdatePlanArgs)
}

// NewPlanHandler creates a new PlanHandler
func NewPlanHandler() *PlanHandler {
	return &PlanHandler{}
}

// Name returns the tool name
func (h *PlanHandler) Name() string {
	return "update_plan"
}

// Handle executes the update_plan tool
func (h *PlanHandler) Handle(arguments string) (*tools.ToolOutput, error) {
	var args tools.UpdatePlanArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse function arguments: %w", err)
	}

	// Validate plan
	if err := tools.ValidatePlanArgs(&args); err != nil {
		return tools.NewToolOutputError(err.Error()), nil
	}

	// Call update callback if set
	if h.OnPlanUpdate != nil {
		h.OnPlanUpdate(&args)
	}

	return tools.NewToolOutput("Plan updated"), nil
}
