package fleet

import (
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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
			wantErr: errors.New(`Interpreter not supported. Bash scripts must run in "#!/bin/sh”.`),
		},
		{
			name:    "valid script",
			script:  "#!/bin/sh\necho 'hello'",
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
