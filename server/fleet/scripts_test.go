package fleet

import (
	"errors"
	"strings"
	"testing"
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
			wantErr: errors.New("The file should be a .sh file."),
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
			err := tt.script.Validate()
			require.Equal(t, tt.wantErr, err)
		})
	}
}

func TestValidateHostScriptContents(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		wantErr error
	}{
		{
			name:    "empty script",
			script:  "",
			wantErr: errors.New("Script contents must not be empty."),
		},
		{
			name:    "too large by byte count",
			script:  strings.Repeat("a", utf8.UTFMax*MaxScriptRuneLen+1),
			wantErr: errors.New("Script is too large. It's limited to 10,000 characters (approximately 125 lines)."),
		},
		{
			name:    "too large by rune count",
			script:  strings.Repeat("🙂", MaxScriptRuneLen+1),
			wantErr: errors.New("Script is too large. It's limited to 10,000 characters (approximately 125 lines)."),
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
			err := ValidateHostScriptContents(tt.script)
			require.Equal(t, tt.wantErr, err)
		})
	}
}
