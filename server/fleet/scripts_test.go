package fleet

import (
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestScriptValidate(t *testing.T) {
	tests := []struct {
		name    string
		script  Script
		wantErr error
	}{
		{
			name: "valid script",
			script: Script{
				Name:           "test.sh",
				ScriptContents: "valid",
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			script: Script{
				Name:           "",
				ScriptContents: "valid",
			},
			wantErr: errors.New("The file name must not be empty."),
		},
		{
			name: "invalid extension",
			script: Script{
				Name:           "test.txt",
				ScriptContents: "valid",
			},
			wantErr: errors.New("File type not supported. Only .sh and .ps1 file type is allowed."),
		},
		{
			name: "invalid script content",
			script: Script{
				Name:           "test.sh",
				ScriptContents: "",
			},
			wantErr: errors.New("Script contents must not be empty."),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.script.ValidateNewScript()
			require.Equal(t, tt.wantErr, err)
		})
	}
}

func TestValidateShebang(t *testing.T) {
	tests := []struct {
		name          string
		contents      string
		directExecute bool
		err           error
	}{
		{
			name:          "no shebang",
			contents:      "echo hi",
			directExecute: false,
		},
		{
			name:          "posix shebang",
			contents:      "#!/bin/sh\necho hi",
			directExecute: true,
		},
		{
			name:          "zsh shebang",
			contents:      "#!/bin/zsh\necho hi",
			directExecute: true,
		},
		{
			name:          "bash shebang",
			contents:      "#!/bin/bash\necho hi",
			directExecute: true,
		},
		{
			name:          "zsh shebang with args",
			contents:      "#!/bin/zsh -x\necho hi",
			directExecute: true,
		},
		{
			name:          "shebang with unsupported interpreter",
			contents:      "#!/usr/bin/python\nprint('hi')",
			directExecute: false,
			err:           ErrUnsupportedInterpreter,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			directExecute, err := ValidateShebang(tc.contents)
			require.Equal(t, tc.directExecute, directExecute)
			require.ErrorIs(t, tc.err, err)
		})
	}
}

func TestValidateHostScriptContents(t *testing.T) {
	tests := []struct {
		name      string
		script    string
		isUnsaved bool
		wantErr   error
	}{
		{
			name:    "empty script",
			script:  "",
			wantErr: errors.New("Script contents must not be empty."),
		},
		{
			name:    "too large by byte count (saved)",
			script:  strings.Repeat("a", utf8.UTFMax*SavedScriptMaxRuneLen+1),
			wantErr: errors.New("Script is too large. It's limited to 500,000 characters (approximately 10,000 lines)."),
		},
		{
			name:      "too large by byte count (unsaved)",
			script:    strings.Repeat("a", utf8.UTFMax*UnsavedScriptMaxRuneLen+1),
			isUnsaved: true,
			wantErr:   errors.New("Script is too large. It's limited to 10,000 characters (approximately 125 lines)."),
		},
		{
			name:    "too large by rune count (saved)",
			script:  strings.Repeat("🙂", SavedScriptMaxRuneLen+1),
			wantErr: errors.New("Script is too large. It's limited to 500,000 characters (approximately 10,000 lines)."),
		},
		{
			name:      "too large by byte count (unsaved)",
			script:    strings.Repeat("a", utf8.UTFMax*UnsavedScriptMaxRuneLen+1),
			isUnsaved: true,
			wantErr:   errors.New("Script is too large. It's limited to 10,000 characters (approximately 125 lines)."),
		},
		{
			name:    "invalid utf8 encoding",
			script:  string([]byte{0xff, 0xfe, 0xfd}),
			wantErr: errors.New("Wrong data format. Only plain text allowed."),
		},
		{
			name:    "unsupported interpreter",
			script:  "#!/bin/bash\necho 'hello'",
			wantErr: ErrUnsupportedInterpreter,
		},
		{
			name:    "valid script",
			script:  "#!/bin/sh\necho 'hello'",
			wantErr: nil,
		},
		{
			name:    "valid zsh script",
			script:  "#!/bin/zsh\necho 'hello'",
			wantErr: nil,
		},
		{
			name:    "valid zsh script",
			script:  "#!/usr/bin/zsh\necho 'hello'",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHostScriptContents(tt.script, !tt.isUnsaved)
			require.Equal(t, tt.wantErr, err)
		})
	}
}

func TestHostTimeout(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name              string
		hostScriptResult  HostScriptResult
		waitForResultTime time.Duration
		expectedResult    bool
	}{
		{
			name: "sync exitcode nil timeout passed",
			hostScriptResult: HostScriptResult{
				SyncRequest: true,
				ExitCode:    nil,
				CreatedAt:   now.Add(-10 * time.Minute),
			},
			waitForResultTime: 5 * time.Minute,
			expectedResult:    true,
		},
		{
			name: "sync exitcode nil timeout not passed",
			hostScriptResult: HostScriptResult{
				SyncRequest: true,
				ExitCode:    nil,
				CreatedAt:   now.Add(-3 * time.Minute),
			},
			waitForResultTime: 5 * time.Minute,
			expectedResult:    false,
		},
		{
			name: "sync exitcode set",
			hostScriptResult: HostScriptResult{
				SyncRequest: true,
				ExitCode:    new(int64),
				CreatedAt:   now.Add(-10 * time.Minute),
			},
			waitForResultTime: 5 * time.Minute,
			expectedResult:    false,
		},
		{
			name: "async exitcode nil",
			hostScriptResult: HostScriptResult{
				SyncRequest: false,
				ExitCode:    nil,
				CreatedAt:   now.Add(-10 * time.Minute),
			},
			waitForResultTime: 5 * time.Minute,
			expectedResult:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.hostScriptResult.HostTimeout(tt.waitForResultTime)
			require.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestUserMessage(t *testing.T) {
	hsr := HostScriptResult{}

	// host timeout true
	m := hsr.UserMessage(true, ptr.Int(0))
	require.Equal(t, RunScriptHostTimeoutErrMsg, m)

	// nil exit code with timeout
	hsr.SyncRequest = true
	hsr.CreatedAt = time.Now().Add(-10 * time.Minute)
	m = hsr.UserMessage(false, ptr.Int(0))
	require.Equal(t, RunScriptHostTimeoutErrMsg, m)

	// nil exit code without timeout
	hsr.SyncRequest = true
	hsr.CreatedAt = time.Now().Add(-3 * time.Minute)
	m = hsr.UserMessage(false, ptr.Int(0))
	require.Equal(t, RunScriptAlreadyRunningErrMsg, m)

	// exit code set, sync request false
	hsr.SyncRequest = false
	m = hsr.UserMessage(false, ptr.Int(0))
	require.Equal(t, RunScriptAsyncScriptEnqueuedMsg, m)

	// use default timeout value in err if 0
	hsr.ExitCode = ptr.Int64(-1)
	m = hsr.UserMessage(false, ptr.Int(0))
	require.Equal(t, "Timeout. Fleet stopped the script after 300 seconds to protect host performance.", m)

	// use default timeout value in err if nil
	m = hsr.UserMessage(false, nil)
	require.Equal(t, "Timeout. Fleet stopped the script after 300 seconds to protect host performance.", m)

	// use provided timeout value in err
	m = hsr.UserMessage(false, ptr.Int(30))
	require.Equal(t, "Timeout. Fleet stopped the script after 30 seconds to protect host performance.", m)

	// run script disabled error
	hsr.ExitCode = ptr.Int64(-2)
	m = hsr.UserMessage(false, ptr.Int(0))
	require.Equal(t, RunScriptDisabledErrMsg, m)
}
