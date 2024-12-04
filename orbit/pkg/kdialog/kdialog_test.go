package kdialog

import (
	"context"
	"os/exec"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/stretchr/testify/assert"
)

type mockExecCmd struct {
	output       []byte
	exitCode     int
	capturedArgs []string
	err          error
}

func (m *mockExecCmd) runWithOutput(args ...string) ([]byte, int, error) {
	m.capturedArgs = append(m.capturedArgs, args...)

	if m.exitCode != 0 {
		return nil, m.exitCode, &exec.ExitError{}
	}

	return m.output, m.exitCode, nil
}

func (m *mockExecCmd) runWithContext(ctx context.Context, args ...string) error {
	m.capturedArgs = append(m.capturedArgs, args...)

	if m.err != nil {
		return m.err
	}

	return nil
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
			expectedArgs: []string{"--password", "Some text", "--title", "A Title"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecCmd{
				output: []byte("some output"),
			}
			k := &KDialog{
				cmdWithOutput: mock.runWithOutput,
			}
			output, err := k.ShowEntry(tt.opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedArgs, mock.capturedArgs)
			assert.Equal(t, []byte("some output"), output)
		})
	}
}

func TestShowEntryError(t *testing.T) {
	mock := &mockExecCmd{
		exitCode: 1,
	}
	k := &KDialog{
		cmdWithOutput: mock.runWithOutput,
	}
	_, err := k.ShowEntry(dialog.EntryOptions{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, dialog.ErrUnknown)
}

func TestShowInfoArgs(t *testing.T) {
	testCases := []struct {
		name         string
		opts         dialog.InfoOptions
		expectedArgs []string
	}{
		{
			name: "Basic Info",
			opts: dialog.InfoOptions{
				Title: "A Title",
				Text:  "Some text",
			},
			expectedArgs: []string{"--msgbox", "Some text", "--title", "A Title"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockExecCmd{}
			k := &KDialog{
				cmdWithContext: mock.runWithContext,
			}
			err := k.ShowInfo(tt.opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedArgs, mock.capturedArgs)
		})
	}
}

// func TestShowProgressArgs(t *testing.T) {
// 	testCases := []struct {
// 		name         string
// 		opts         dialog.ProgressOptions
// 		expectedArgs []string
// 	}{
// 		{
// 			name: "Basic Progress",
// 			opts: dialog.ProgressOptions{
// 				Title: "A Title",
// 				Text:  "Some text",
// 			},
// 			expectedArgs: []string{"--progressbar", "Some text", "--title", "A Title"},
// 		},
// 	}

// 	for _, tt := range testCases {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mock := &mockExecCmd{
// 				output: []byte("org.kde.kdialog.ProgressDialog /Progress_1"),
// 			}
// 			k := &KDialog{
// 				cmdWithOutput: mock.runWithOutput,
// 			}
// 			_, err := k.ShowProgress(tt.opts)
// 			assert.NoError(t, err)
// 			assert.Equal(t, tt.expectedArgs, mock.capturedArgs)
// 		})
// 	}
// }
