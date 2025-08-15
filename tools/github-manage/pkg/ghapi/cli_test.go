package ghapi

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
)

func TestRunCommandAndReturnOutput(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		expectError bool
		expectOut   string
	}{
		{
			name:        "simple echo command",
			command:     "echo hello",
			expectError: false,
			expectOut:   "hello",
		},
		{
			name:        "command with output",
			command:     "echo 'test output'",
			expectError: false,
			expectOut:   "test output",
		},
		{
			name:        "invalid command",
			command:     "nonexistentcommand12345",
			expectError: true,
			expectOut:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip certain tests on Windows since bash might not be available
			if runtime.GOOS == "windows" && !tt.expectError {
				t.Skip("Skipping bash command test on Windows")
			}

			output, err := RunCommandAndReturnOutput(tt.command)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			outputStr := strings.TrimSpace(string(output))
			if !strings.Contains(outputStr, tt.expectOut) {
				t.Errorf("Expected output to contain '%s', got '%s'", tt.expectOut, outputStr)
			}
		})
	}
}

func TestRunCommandAndReturnOutput_ErrorHandling(t *testing.T) {
	// Test that stderr is captured when command fails
	output, err := RunCommandAndReturnOutput("bash -c 'echo error >&2; exit 1'")

	if err == nil {
		t.Error("Expected error for failing command")
	}

	if output != nil {
		t.Error("Expected nil output for failing command")
	}
}

func TestRunCommandAndReturnOutput_EmptyCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping bash command test on Windows")
	}

	output, err := RunCommandAndReturnOutput("")
	if err != nil {
		t.Errorf("Empty command should not error, got: %v", err)
	}

	if !bytes.Equal(output, []byte{}) && !bytes.Equal(output, []byte("\n")) {
		t.Errorf("Expected empty or newline output, got: %q", output)
	}
}
