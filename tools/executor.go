package tools

import (
	"bytes"
	"devtool/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func ExecuteTool(tool config.ToolConfig, args map[string]interface{}) (string, error) {
	if tool.Type == "shell" {
		return executeShellTool(tool, args)
	}
	// Default to HTTP
	return executeHTTPTool(tool, args)
}

func executeShellTool(tool config.ToolConfig, args map[string]interface{}) (string, error) {
	// We run the command using sh -c to allow for complex commands
	cmd := exec.Command("sh", "-c", tool.Command)

	// Prepare environment variables
	env := os.Environ()
	for k, v := range args {
		// Convert value to string
		valStr := fmt.Sprintf("%v", v)
		// Sanitize key to be upper case and replace - with _
		key := strings.ToUpper(strings.ReplaceAll(k, "-", "_"))
		env = append(env, fmt.Sprintf("%s=%s", key, valStr))
	}
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command execution failed: %w", err)
	}

	return string(output), nil
}

func executeHTTPTool(tool config.ToolConfig, args map[string]interface{}) (string, error) {
	// 1. Prepare URL
	// Simple implementation: assume URL doesn't need path param substitution for now,
	// or we could use a library for that. Let's stick to simple.

	// 2. Prepare Body or Query
	var bodyReader io.Reader
	if tool.Method == "POST" || tool.Method == "PUT" || tool.Method == "PATCH" {
		jsonBody, err := json.Marshal(args)
		if err != nil {
			return "", fmt.Errorf("failed to marshal args: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(tool.Method, tool.URL, bodyReader)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 3. Set Headers
	for k, v := range tool.Headers {
		// Expand env vars in headers
		expandedVal := os.ExpandEnv(v)
		req.Header.Set(k, expandedVal)
	}
	req.Header.Set("Content-Type", "application/json")

	// 4. Query params for GET
	if tool.Method == "GET" {
		q := req.URL.Query()
		for k, v := range args {
			q.Add(k, fmt.Sprintf("%v", v))
		}
		req.URL.RawQuery = q.Encode()
	}

	// 5. Execute
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return string(respBody), fmt.Errorf("server returned error status: %d", resp.StatusCode)
	}

	return string(respBody), nil
}
