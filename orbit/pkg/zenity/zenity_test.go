package zenity

import (
	"context"
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
	waitDuration time.Duration
}

// MockCommandContext simulates exec.CommandContext and captures arguments
func (m *mockExecCmd) runWithOutput(ctx context.Context, args ...string) ([]byte, int, error) {
	m.capturedArgs = append(m.capturedArgs, args...)

	if m.exitCode != 0 {
		return nil, m.exitCode, &exec.ExitError{}
	}

	return m.output, m.exitCode, nil
}

func (m *mockExecCmd) runWithWait(ctx context.Context, args ...string) error {
	m.capturedArgs = append(m.capturedArgs, args...)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(m.waitDuration):

	}

	return nil
}

func TestShowEntryArgs(t *testing.T) {
	ctx := context.Background()

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
			output, err := z.ShowEntry(ctx, dialog.EntryOptions{})
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
		cmdWithOutput: mock.runWithOutput,
	}
	output, err := z.ShowEntry(ctx, dialog.EntryOptions{})
	assert.NoError(t, err)
	assert.Equal(t, []byte("some output"), output)
}

func TestShowInfoArgs(t *testing.T) {
	ctx := context.Background()

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
			err := z.ShowInfo(ctx, dialog.InfoOptions{})
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestProgressArgs(t *testing.T) {
	ctx := context.Background()

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
			expectedArgs: []string{"--progress", "--title=A Title", "--text=Some text", "--pulsate", "--no-cancel"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecCmd{}
			z := &Zenity{
				cmdWithWait: mock.runWithWait,
			}
			err := z.ShowProgress(ctx, tt.opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedArgs, mock.capturedArgs)
		})
	}
}

func TestProgressKillOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mock := &mockExecCmd{
		waitDuration: 5 * time.Second,
	}
	z := &Zenity{
		cmdWithWait: mock.runWithWait,
	}

	done := make(chan struct{})
	start := time.Now()

	go func() {
		_ = z.ShowProgress(ctx, dialog.ProgressOptions{})
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	assert.True(t, time.Since(start) < 5*time.Second)
}
