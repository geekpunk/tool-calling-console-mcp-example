package tools

import (
	"devtool/config"
	"fmt"
	"strings"
)

// ExecuteWorkflow executes a defined workflow.
func ExecuteWorkflow(wf config.WorkflowConfig, tools []config.ToolConfig, globalArgs map[string]interface{}) (string, error) {
	// Store outputs from each step
	outputs := make(map[string]string)
	
	// Add global args to outputs for substitution (e.g. {{args.ip}})
	// We flat map them for now? Or keep under "args"?
	// Let's use "input" namespace.
	
	for _, step := range wf.Steps {
		// Find the tool
		var tool *config.ToolConfig
		for _, t := range tools {
			if t.Name == step.Tool {
				tool = &t
				break
			}
		}
		if tool == nil {
			return "", fmt.Errorf("tool '%s' not found for step '%s'", step.Tool, step.Name)
		}

		// Prepare arguments
		stepArgs := make(map[string]interface{})
		for k, v := range step.Args {
			// Resolve templates in values
			valStr, ok := v.(string)
			if ok {
				// Simple substitution: {{stepName}} with its output
				// and {{input.argName}} with global args
				
				// 1. Inputs
				for argKey, argVal := range globalArgs {
					placeholder := fmt.Sprintf("{{input.%s}}", argKey)
					if strings.Contains(valStr, placeholder) {
						valStr = strings.ReplaceAll(valStr, placeholder, fmt.Sprintf("%v", argVal))
					}
				}

				// 2. Previous step outputs
				for stepName, stepOutput := range outputs {
					placeholder := fmt.Sprintf("{{%s}}", stepName)
					if strings.Contains(valStr, placeholder) {
						valStr = strings.ReplaceAll(valStr, placeholder, stepOutput)
					}
				}
				stepArgs[k] = valStr
			} else {
				stepArgs[k] = v
			}
		}

		// Execute tool
		out, err := ExecuteTool(*tool, stepArgs)
		if err != nil {
			return "", fmt.Errorf("step '%s' failed: %w. Output: %s", step.Name, err, out)
		}
		
		// Trim whitespace for cleaner substitution
		outputs[step.Name] = strings.TrimSpace(out)
	}

	// Format final output
	result := wf.Output
	if result == "" {
		// Default to dumping all steps
		var builder strings.Builder
		for k, v := range outputs {
			builder.WriteString(fmt.Sprintf("%s: %s\n", k, v))
		}
		return builder.String(), nil
	}

	// Replace placeholders in output template
	for stepName, stepOutput := range outputs {
		placeholder := fmt.Sprintf("{{%s}}", stepName)
		if strings.Contains(result, placeholder) {
			result = strings.ReplaceAll(result, placeholder, stepOutput)
		}
	}
	
	for argKey, argVal := range globalArgs {
		placeholder := fmt.Sprintf("{{input.%s}}", argKey)
		if strings.Contains(result, placeholder) {
			result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", argVal))
		}
	}

	return result, nil
}
