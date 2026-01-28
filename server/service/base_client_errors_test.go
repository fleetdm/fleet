package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractServerErrorText(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "valid JSON error",
			body:     `{"message": "Something went wrong", "errors": [{"name": "error1", "reason": "invalid input"}]}`,
			expected: "Something went wrong: invalid input",
		},
		{
			name:     "403 Forbidden HTML",
			body:     `<!DOCTYPE html><html><head><title>403 Forbidden</title></head><body><h1>403 Forbidden</h1><p>You don't have permission to access this resource.</p></body></html>`,
			expected: "server returned HTML instead of JSON response, body: <!DOCTYPE html><html><head><title>403 Forbidden</title></head><body><h1>403 Forbidden</h1><p>You don't have permission to access this resource.</p></body></html>",
		},
		{
			name:     "HTML with uppercase tags",
			body:     `<HTML><HEAD><TITLE>Error</TITLE></HEAD><BODY>Server Error</BODY></HTML>`,
			expected: "server returned HTML instead of JSON response, body: <HTML><HEAD><TITLE>Error</TITLE></HEAD><BODY>Server Error</BODY></HTML>",
		},
		{
			name:     "long HTML gets truncated",
			body:     `<!DOCTYPE html><html><head><title>Error Page</title></head><body>` + strings.Repeat("A", 200) + `</body></html>`,
			expected: "server returned HTML instead of JSON response, body: <!DOCTYPE html><html><head><title>Error Page</title></head><body>" + strings.Repeat("A", 135) + "...",
		},
		{
			name:     "plain text error",
			body:     "Connection refused",
			expected: "Connection refused",
		},
		{
			name:     "empty response",
			body:     "",
			expected: "empty response body",
		},
		{
			name:     "long plain text truncated",
			body:     strings.Repeat("a", 250),
			expected: strings.Repeat("a", 200) + "...",
		},
		{
			name:     "invalid JSON",
			body:     `{invalid json}`,
			expected: "{invalid json}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.body)
			result := extractServerErrorText(reader)
			assert.Equal(t, tt.expected, result)
		})
	}
}
