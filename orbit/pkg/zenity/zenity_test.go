package zenity

import (
	"os/exec"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

type mockExecCmd struct {
	output       []byte
	exitCode     int
	capturedArgs []string
}

// MockCommandContext simulates exec.CommandContext and captures arguments
func (m *mockExecCmd) runWithOutput(args ...string) ([]byte, int, error) {
	m.capturedArgs = append(m.capturedArgs, args...)

	if m.exitCode != 0 {
		return nil, m.exitCode, &exec.ExitError{}
	}

	return m.output, m.exitCode, nil
}

func (m *mockExecCmd) runWithStdin(args ...string) (func() error, error) {
	m.capturedArgs = append(m.capturedArgs, args...)

	return nil, nil
}

func TestShowEntryArgs(t *testing.T) {
	testCases := []struct {
		name         string
		opts         dialog.EntryOptions
		expectedArgs []string
	}{
		{
			name: "Basic Entry",
			opts: dialog.EntryOptions{
				Title: "A Title",
				Text:  "Some text",
			},
			expectedArgs: []string{"--entry", "--title=A Title", "--text=Some text"},
		},
		{
			name: "All Options",
			opts: dialog.EntryOptions{
				Title:    "Another Title",
				Text:     "Some more text",
				HideText: true,
				TimeOut:  1 * time.Minute,
			},
			expectedArgs: []string{"--entry", "--title=Another Title", "--text=Some more text", "--hide-text", "--timeout=60"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecCmd{
				output: []byte("some output"),
			}
			z := &Zenity{
				cmdWithOutput: mock.runWithOutput,
			}
			output, err := z.ShowEntry(tt.opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedArgs, mock.capturedArgs)
			assert.Equal(t, []byte("some output"), output)
		})
	}
}

func TestShowEntryError(t *testing.T) {
	testcases := []struct {
		name        string
		exitCode    int
		expectedErr error
	}{
		{
			name:        "Dialog Cancelled",
			exitCode:    1,
			expectedErr: dialog.ErrCanceled,
		},
		{
			name:        "Dialog Timed Out",
			exitCode:    5,
			expectedErr: dialog.ErrTimeout,
		},
		{
			name:        "Unknown Error",
			exitCode:    99,
			expectedErr: dialog.ErrUnknown,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecCmd{
				exitCode: tt.exitCode,
			}
			z := &Zenity{
				cmdWithOutput: mock.runWithOutput,
			}
			output, err := z.ShowEntry(dialog.EntryOptions{})
			require.ErrorIs(t, err, tt.expectedErr)
			assert.Nil(t, output)
		})
	}
}

func TestShowEntrySuccess(t *testing.T) {
	mock := &mockExecCmd{
		output: []byte("some output"),
	}
	z := &Zenity{
		cmdWithOutput: mock.runWithOutput,
	}
	output, err := z.ShowEntry(dialog.EntryOptions{})
	assert.NoError(t, err)
	assert.Equal(t, []byte("some output"), output)
}

func TestShowInfoArgs(t *testing.T) {
	testCases := []struct {
		name         string
		opts         dialog.InfoOptions
		expectedArgs []string
	}{
		{
			name:         "Basic Entry",
			opts:         dialog.InfoOptions{},
			expectedArgs: []string{"--info"},
		},
		{
			name: "All Options",
			opts: dialog.InfoOptions{
				Title:   "Another Title",
				Text:    "Some more text",
				TimeOut: 1 * time.Minute,
			},
			expectedArgs: []string{"--info", "--title=Another Title", "--text=Some more text", "--timeout=60"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecCmd{}
			z := &Zenity{
				cmdWithOutput: mock.runWithOutput,
			}
			err := z.ShowInfo(tt.opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedArgs, mock.capturedArgs)
		})
	}
}

func TestShowInfoError(t *testing.T) {
	testcases := []struct {
		name        string
		exitCode    int
		expectedErr error
	}{
		{
			name:        "Dialog Timed Out",
			exitCode:    5,
			expectedErr: dialog.ErrTimeout,
		},
		{
			name:        "Unknown Error",
			exitCode:    99,
			expectedErr: dialog.ErrUnknown,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecCmd{
				exitCode: tt.exitCode,
			}
			z := &Zenity{
				cmdWithOutput: mock.runWithOutput,
			}
			err := z.ShowInfo(dialog.InfoOptions{})
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestProgressArgs(t *testing.T) {
	testCases := []struct {
		name         string
		opts         dialog.ProgressOptions
		expectedArgs []string
	}{
		{
			name: "Basic Entry",
			opts: dialog.ProgressOptions{
				Title: "A Title",
				Text:  "Some text",
			},
			expectedArgs: []string{"--progress", "--title=A Title", "--text=Some text", "--pulsate", "--no-cancel", "--auto-close"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecCmd{}
			z := &Zenity{
				cmdWithCancel: mock.runWithStdin,
			}
			_, err := z.ShowProgress(tt.opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedArgs, mock.capturedArgs)
		})
	}
}
