package handlers

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/codex-fork/gosdk/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanHandler_BasicUpdate(t *testing.T) {
	handler := NewPlanHandler()

	args := tools.UpdatePlanArgs{
		Explanation: "Starting implementation",
		Plan: []tools.PlanItem{
			{Step: "Research existing code", Status: "completed"},
			{Step: "Implement feature", Status: "in_progress"},
			{Step: "Write tests", Status: "pending"},
		},
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.True(t, *output.Success)
	assert.Equal(t, "Plan updated", output.Content)
}

func TestPlanHandler_WithCallback(t *testing.T) {
	handler := NewPlanHandler()

	var capturedArgs *tools.UpdatePlanArgs
	handler.OnPlanUpdate = func(args *tools.UpdatePlanArgs) {
		capturedArgs = args
	}

	args := tools.UpdatePlanArgs{
		Explanation: "Test callback",
		Plan: []tools.PlanItem{
			{Step: "Step 1", Status: "pending"},
		},
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.True(t, *output.Success)

	// Verify callback was called
	assert.NotNil(t, capturedArgs)
	assert.Equal(t, "Test callback", capturedArgs.Explanation)
	assert.Len(t, capturedArgs.Plan, 1)
}

func TestPlanHandler_EmptyPlan(t *testing.T) {
	handler := NewPlanHandler()

	args := tools.UpdatePlanArgs{
		Plan: []tools.PlanItem{},
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.True(t, *output.Success)
}

func TestPlanHandler_NoExplanation(t *testing.T) {
	handler := NewPlanHandler()

	args := tools.UpdatePlanArgs{
		Plan: []tools.PlanItem{
			{Step: "Do something", Status: "pending"},
		},
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.True(t, *output.Success)
}

func TestPlanHandler_InvalidJSON(t *testing.T) {
	handler := NewPlanHandler()

	_, err := handler.Handle("invalid json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse function arguments")
}

func TestPlanHandler_Name(t *testing.T) {
	handler := NewPlanHandler()
	assert.Equal(t, "update_plan", handler.Name())
}

func TestValidatePlanArgs(t *testing.T) {
	// Test with valid plan
	args := &tools.UpdatePlanArgs{
		Plan: []tools.PlanItem{
			{Step: "Step 1", Status: "completed"},
			{Step: "Step 2", Status: "in_progress"},
			{Step: "Step 3", Status: "pending"},
		},
	}
	err := tools.ValidatePlanArgs(args)
	assert.NoError(t, err)

	// Test with multiple in_progress (validation is soft, doesn't error)
	argsMultiple := &tools.UpdatePlanArgs{
		Plan: []tools.PlanItem{
			{Step: "Step 1", Status: "in_progress"},
			{Step: "Step 2", Status: "in_progress"},
		},
	}
	err = tools.ValidatePlanArgs(argsMultiple)
	assert.NoError(t, err) // Soft validation
}

func TestPlanItem_Statuses(t *testing.T) {
	statuses := []string{"pending", "in_progress", "completed"}

	for _, status := range statuses {
		item := tools.PlanItem{
			Step:   "Test step",
			Status: status,
		}
		assert.Equal(t, status, item.Status)
	}
}

func TestPlanHandler_AllStatuses(t *testing.T) {
	handler := NewPlanHandler()

	args := tools.UpdatePlanArgs{
		Plan: []tools.PlanItem{
			{Step: "Completed step", Status: "completed"},
			{Step: "In progress step", Status: "in_progress"},
			{Step: "Pending step", Status: "pending"},
		},
	}
	argsJSON, _ := json.Marshal(args)

	output, err := handler.Handle(string(argsJSON))
	require.NoError(t, err)
	assert.True(t, *output.Success)
}
