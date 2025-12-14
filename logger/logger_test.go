package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	// Create temp dir
	dir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	logPath := filepath.Join(dir, "test.log")

	// Setup logger
	err = Setup(logPath)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Write logs
	Info("This is an info message")
	Error("This is an error message: %d", 500)

	// Read file content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	output := string(content)

	// Verify content
	if !strings.Contains(output, "[INFO] This is an info message") {
		t.Errorf("Expected info log not found. Got:\n%s", output)
	}
	if !strings.Contains(output, "[ERROR] This is an error message: 500") {
		t.Errorf("Expected error log not found. Got:\n%s", output)
	}
}
