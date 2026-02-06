package service

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeBase64Script(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "simple string",
			input:    base64.StdEncoding.EncodeToString([]byte("Hello World")),
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name:     "powershell with dollar brace",
			input:    base64.StdEncoding.EncodeToString([]byte("${env:TEMP}")),
			expected: "${env:TEMP}",
			wantErr:  false,
		},
		{
			name:     "powershell install script pattern",
			input:    base64.StdEncoding.EncodeToString([]byte("$installProcess = Start-Process msiexec.exe")),
			expected: "$installProcess = Start-Process msiexec.exe",
			wantErr:  false,
		},
		{
			name:     "multiline powershell script",
			input:    base64.StdEncoding.EncodeToString([]byte("$logFile = \"${env:TEMP}/fleet-install.log\"\nStart-Process msiexec.exe")),
			expected: "$logFile = \"${env:TEMP}/fleet-install.log\"\nStart-Process msiexec.exe",
			wantErr:  false,
		},
		{
			name:     "unicode characters",
			input:    base64.StdEncoding.EncodeToString([]byte("echo \"Hello 世界\"")),
			expected: "echo \"Hello 世界\"",
			wantErr:  false,
		},
		{
			name:     "invalid base64",
			input:    "not-valid-base64!!!",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "truncated base64",
			input:    "SGVsbG8gV29ybGQ", // missing padding
			expected: "",
			wantErr:  true, // Go's StdEncoding requires proper padding
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeBase64Script(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIsScriptsEncoded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		header   string
		expected bool
	}{
		{
			name:     "header set to base64",
			header:   "base64",
			expected: true,
		},
		{
			name:     "header not set",
			header:   "",
			expected: false,
		},
		{
			name:     "header set to different value",
			header:   "gzip",
			expected: false,
		},
		{
			name:     "header set to Base64 (wrong case)",
			header:   "Base64",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			if tt.header != "" {
				req.Header.Set(ScriptsEncodedHeader, tt.header)
			}
			result := isScriptsEncoded(req)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestScriptsEncodedHeader(t *testing.T) {
	// Verify the header constant has the expected value
	require.Equal(t, "X-Fleet-Scripts-Encoded", ScriptsEncodedHeader)
}
