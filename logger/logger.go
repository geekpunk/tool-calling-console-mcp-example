package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	logFile io.Writer
	mu      sync.Mutex
)

// Setup initializes the logger with an output file.
// It writes to both Stderr and the file.
func Setup(path string) error {
	mu.Lock()
	defer mu.Unlock()

	if path == "" {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	
	logFile = f
	return nil
}

func write(level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	// Trim newline to avoid double spacing if the message already has one, logic below handles it
	msg = strings.TrimRight(msg, "\n")
	
	formatted := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, msg)

	mu.Lock()
	defer mu.Unlock()

	// Always write to Stderr
	os.Stderr.Write([]byte(formatted))

	// Write to file if configured
	if logFile != nil {
		logFile.Write([]byte(formatted))
	}
}

func Info(format string, args ...interface{}) {
	write("INFO", format, args...)
}

func Error(format string, args ...interface{}) {
	write("ERROR", format, args...)
}
