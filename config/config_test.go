package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory
	dir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create a sample config file
	configContent := `
logfile: "test.log"
server:
  port: 8080
tools:
  - name: "test-tool"
    type: "shell"
    command: "echo hello"
    description: "A test tool"
workflows:
  - name: "test-workflow"
    description: "A test workflow"
    steps:
      - name: "step1"
        tool: "test-tool"
`
	configPath := filepath.Join(dir, "devtool.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Load the config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify values
	if cfg.LogFile != "test.log" {
		t.Errorf("Expected LogFile 'test.log', got '%s'", cfg.LogFile)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected Server.Port 8080, got %d", cfg.Server.Port)
	}
	if len(cfg.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(cfg.Tools))
	}
	if cfg.Tools[0].Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", cfg.Tools[0].Name)
	}
	if len(cfg.Workflows) != 1 {
		t.Fatalf("Expected 1 workflow, got %d", len(cfg.Workflows))
	}
	if cfg.Workflows[0].Name != "test-workflow" {
		t.Errorf("Expected workflow name 'test-workflow', got '%s'", cfg.Workflows[0].Name)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("non-existent-file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// Create a temporary directory
	dir, err := os.MkdirTemp("", "config_test_invalid")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create an invalid config file
	configPath := filepath.Join(dir, "invalid.yaml")
	err = os.WriteFile(configPath, []byte("invalid: [ yaml"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}
