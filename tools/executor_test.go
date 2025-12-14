package tools

import (
	"devtool/config"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExecuteTool_Shell(t *testing.T) {
	tool := config.ToolConfig{
		Name:    "echo-tool",
		Type:    "shell",
		Command: "echo $MESSAGE", // use $MESSAGE to test env var passing
	}

	args := map[string]interface{}{
		"message": "Hello World",
	}

	output, err := ExecuteTool(tool, args)
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	expected := "Hello World\n"
	if output != expected {
		t.Errorf("Expected output '%s', got '%s'", expected, output)
	}
}

func TestExecuteTool_Shell_Error(t *testing.T) {
	tool := config.ToolConfig{
		Name:    "fail-tool",
		Type:    "shell",
		Command: "exit 1",
	}

	_, err := ExecuteTool(tool, map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for failing command, got nil")
	}
}

func TestExecuteTool_HTTP_GET(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected method GET, got %s", r.Method)
		}
		query := r.URL.Query().Get("q")
		if query != "test" {
			t.Errorf("Expected query param 'q' to be 'test', got '%s'", query)
		}
		fmt.Fprint(w, "success")
	}))
	defer ts.Close()

	tool := config.ToolConfig{
		Name:   "http-get",
		Type:   "http",
		URL:    ts.URL,
		Method: "GET",
	}

	args := map[string]interface{}{
		"q": "test",
	}

	output, err := ExecuteTool(tool, args)
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if output != "success" {
		t.Errorf("Expected output 'success', got '%s'", output)
	}
}

func TestExecuteTool_HTTP_POST(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "test" {
			t.Errorf("Expected body param 'name' to be 'test', got '%v'", body["name"])
		}
		fmt.Fprint(w, "created")
	}))
	defer ts.Close()

	tool := config.ToolConfig{
		Name:   "http-post",
		Type:   "http",
		URL:    ts.URL,
		Method: "POST",
	}

	args := map[string]interface{}{
		"name": "test",
	}

	output, err := ExecuteTool(tool, args)
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}

	if output != "created" {
		t.Errorf("Expected output 'created', got '%s'", output)
	}
}

func TestExecuteTool_HTTP_Headers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test") != "test-value" {
			t.Errorf("Expected X-Test header 'test-value', got '%s'", r.Header.Get("X-Test"))
		}
		fmt.Fprint(w, "ok")
	}))
	defer ts.Close()

	tool := config.ToolConfig{
		Name:   "http-headers",
		Type:   "http",
		URL:    ts.URL,
		Method: "GET",
		Headers: map[string]string{
			"X-Test": "test-value",
		},
	}

	_, err := ExecuteTool(tool, map[string]interface{}{})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}
}

func TestExecuteTool_HTTP_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "error")
	}))
	defer ts.Close()

	tool := config.ToolConfig{
		Name:   "http-error",
		Type:   "http",
		URL:    ts.URL,
		Method: "GET",
	}

	_, err := ExecuteTool(tool, map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Expected error to contain 500, got %v", err)
	}
}
