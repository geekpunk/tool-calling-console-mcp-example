package tools

import (
	"devtool/config"
	"strings"
	"testing"
)

func TestExecuteWorkflow(t *testing.T) {
	// Define tools
	tools := []config.ToolConfig{
		{
			Name:    "echo-tool",
			Type:    "shell",
			Command: "printf '%s' \"$TEXT\"",
		},
		{
			Name:    "reverse-tool",
			Type:    "shell",
			Command: "printf '%s' \"$INPUT\" | rev",
		},
	}

	// Define workflow
	wf := config.WorkflowConfig{
		Name: "test-wf",
		Steps: []config.StepConfig{
			{
				Name: "step1",
				Tool: "echo-tool",
				Args: map[string]interface{}{
					"text": "{{input.start}}",
				},
			},
			{
				Name: "step2",
				Tool: "reverse-tool",
				Args: map[string]interface{}{
					"input": "{{step1}}",
				},
			},
		},
		Output: "Result: {{step2}}",
	}

	globalArgs := map[string]interface{}{
		"start": "hello",
	}

	output, err := ExecuteWorkflow(wf, tools, globalArgs)
	if err != nil {
		t.Fatalf("ExecuteWorkflow failed: %v", err)
	}

	expected := "Result: olleh"
	if strings.TrimSpace(output) != expected {
		t.Errorf("Expected output '%s', got '%s'", expected, output)
	}
}

func TestExecuteWorkflow_MissingTool(t *testing.T) {
	tools := []config.ToolConfig{}
	wf := config.WorkflowConfig{
		Name: "fail-wf",
		Steps: []config.StepConfig{
			{
				Name: "step1",
				Tool: "missing-tool",
			},
		},
	}

	_, err := ExecuteWorkflow(wf, tools, map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for missing tool, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got %v", err)
	}
}

func TestExecuteWorkflow_StepFailure(t *testing.T) {
	tools := []config.ToolConfig{
		{
			Name:    "fail-tool",
			Type:    "shell",
			Command: "exit 1",
		},
	}
	wf := config.WorkflowConfig{
		Name: "fail-wf",
		Steps: []config.StepConfig{
			{
				Name: "step1",
				Tool: "fail-tool",
			},
		},
	}

	_, err := ExecuteWorkflow(wf, tools, map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for failing step, got nil")
	}
}
