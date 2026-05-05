package logger

import (
	"bytes"
	"testing"
	"strings"
)

func TestLogger(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Initialize logger with test buffer
	if err := Init(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Override the output to use our buffer for testing
	SetOutput(&buf)

	// Test various log levels
	Info("This is an info message")
	Infof("This is an info message with value: %d", 42)
	Error("This is an error message")
	Errorf("This is an error message with error: %v", "test error")
	Debug("This is a debug message")
	Debugf("This is a debug message with data: %s", "test data")

	// Check that output was written
	output := buf.String()
	if len(output) == 0 {
		t.Error("No output was written to the logger")
	}

	// Check that all message types are present
	if !strings.Contains(output, "INFO:") {
		t.Error("INFO message not found in output")
	}
	if !strings.Contains(output, "ERROR:") {
		t.Error("ERROR message not found in output")
	}
	if !strings.Contains(output, "DEBUG:") {
		t.Error("DEBUG message not found in output")
	}

	// Check specific content
	if !strings.Contains(output, "This is an info message") {
		t.Error("Info message content not found")
	}
	if !strings.Contains(output, "value: 42") {
		t.Error("Formatted info message content not found")
	}
}
