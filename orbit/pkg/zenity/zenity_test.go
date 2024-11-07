package zenity

import (
	"context"
	"os/exec"
	"slices"
	"testing"
	"time"
)

// Variables to capture the command and args for verification
var capturedArgs []string

// MockCommandContext simulates exec.CommandContext and captures arguments
func MockCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	capturedArgs = append([]string{name}, args...)
	return exec.CommandContext(ctx, name, args...) // Return a dummy Cmd
}

// MockProcessState simulates the ProcessState with a specific exit code.
type MockProcessState struct {
	exitCode int
}

func (m *MockProcessState) ExitCode() int {
	return m.exitCode
}

func TestShowEntryArgs(t *testing.T) {
	ctx := context.Background()
	z := &Zenity{
		CommandContext: MockCommandContext,
	}

	testCases := []struct {
		name         string
		opts         EntryOptions
		expectedArgs []string
	}{
		{
			name: "Basic Entry",
			opts: EntryOptions{
				Title: "A Title",
				Text:  "Some text",
			},
			expectedArgs: []string{"zenity", "--entry", `--title="A Title"`, `--text="Some text"`},
		},
		{
			name: "All Options",
			opts: EntryOptions{
				Title:    "Another Title",
				Text:     "Some more text",
				HideText: true,
				TimeOut:  1 * time.Minute,
			},
			expectedArgs: []string{"zenity", "--entry", `--title="Another Title"`, `--text="Some more text"`, "--hide-text", "--timeout=60"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			capturedArgs = nil // Reset capturedArgs before each test
			z.ShowEntry(ctx, tt.opts)
			if !slices.Equal(capturedArgs, tt.expectedArgs) {
				t.Errorf("expected args %v, got %v", tt.expectedArgs, capturedArgs)
			}
		})
	}
}
