//go:build linux

package zenity

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

type mockExecCmd struct {
	output       []byte
	exitCode     int
	capturedArgs []string
}

// MockCommandContext simulates exec.CommandContext and captures arguments
func (m *mockExecCmd) run(ctx context.Context, args ...string) ([]byte, int, error) {
	m.capturedArgs = append(m.capturedArgs, args...)

	if m.exitCode != 0 {
		return nil, m.exitCode, &exec.ExitError{}
	}

	return m.output, m.exitCode, nil
}

func TestShowEntryArgs(t *testing.T) {
	ctx := context.Background()

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
			expectedArgs: []string{"--entry", `--title="A Title"`, `--text="Some text"`},
		},
		{
			name: "All Options",
			opts: EntryOptions{
				Title:    "Another Title",
				Text:     "Some more text",
				HideText: true,
				TimeOut:  1 * time.Minute,
			},
			expectedArgs: []string{"--entry", `--title="Another Title"`, `--text="Some more text"`, "--hide-text", "--timeout=60"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecCmd{
				output: []byte("some output"),
			}
			z := &Zenity{
				execCmdFn: mock.run,
			}
			output, err := z.ShowEntry(ctx, tt.opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedArgs, mock.capturedArgs)
			assert.Equal(t, []byte("some output"), output)
		})
	}
}

func TestShowEntryError(t *testing.T) {
	ctx := context.Background()

	testcases := []struct {
		name        string
		exitCode    int
		expectedErr error
	}{
		{
			name:        "Dialog Cancelled",
			exitCode:    1,
			expectedErr: ErrCanceled,
		},
		{
			name:        "Dialog Timed Out",
			exitCode:    5,
			expectedErr: ErrTimeout,
		},
		{
			name:        "Unknown Error",
			exitCode:    99,
			expectedErr: ErrUnknown,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecCmd{
				exitCode: tt.exitCode,
			}
			z := &Zenity{
				execCmdFn: mock.run,
			}
			output, err := z.ShowEntry(ctx, EntryOptions{})
			require.ErrorIs(t, err, tt.expectedErr)
			assert.Nil(t, output)
		})
	}
}

func TestShowEntrySuccess(t *testing.T) {
	ctx := context.Background()

	mock := &mockExecCmd{
		output: []byte("some output"),
	}
	z := &Zenity{
		execCmdFn: mock.run,
	}
	output, err := z.ShowEntry(ctx, EntryOptions{})
	assert.NoError(t, err)
	assert.Equal(t, []byte("some output"), output)
}

func TestShowInfoArgs(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		opts         InfoOptions
		expectedArgs []string
	}{
		{
			name:         "Basic Entry",
			opts:         InfoOptions{},
			expectedArgs: []string{"--info"},
		},
		{
			name: "All Options",
			opts: InfoOptions{
				Title:   "Another Title",
				Text:    "Some more text",
				TimeOut: 1 * time.Minute,
			},
			expectedArgs: []string{"--info", `--title="Another Title"`, `--text="Some more text"`, "--timeout=60"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecCmd{}
			z := &Zenity{
				execCmdFn: mock.run,
			}
			err := z.ShowInfo(ctx, tt.opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedArgs, mock.capturedArgs)
		})
	}
}

func TestShowInfoError(t *testing.T) {
	ctx := context.Background()

	testcases := []struct {
		name        string
		exitCode    int
		expectedErr error
	}{
		{
			name:        "Dialog Timed Out",
			exitCode:    5,
			expectedErr: ErrTimeout,
		},
		{
			name:        "Unknown Error",
			exitCode:    99,
			expectedErr: ErrUnknown,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecCmd{
				exitCode: tt.exitCode,
			}
			z := &Zenity{
				execCmdFn: mock.run,
			}
			err := z.ShowInfo(ctx, InfoOptions{})
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
