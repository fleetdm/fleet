package notarize

import (
	"errors"
	"testing"
)

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-network error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "Code=-1009 offline error",
			err:      errors.New("Error: HTTPError(statusCode: nil, error: Error Domain=NSURLErrorDomain Code=-1009 \"The Internet connection appears to be offline.\")"),
			expected: true,
		},
		{
			name:     "Code=-1001 timeout error",
			err:      errors.New("Error Domain=NSURLErrorDomain Code=-1001 \"The request timed out.\""),
			expected: true,
		},
		{
			name:     "Code=-1004 connection error",
			err:      errors.New("Error Domain=NSURLErrorDomain Code=-1004 \"Could not connect to the server.\""),
			expected: true,
		},
		{
			name:     "Code=-1005 network lost",
			err:      errors.New("Error Domain=NSURLErrorDomain Code=-1005 \"The network connection was lost.\""),
			expected: true,
		},
		{
			name:     "Code=-19000 structured error",
			err:      Errors{{Code: -19000, Message: "Network became unavailable"}},
			expected: true,
		},
		{
			name:     "structured error with non-network code",
			err:      Errors{{Code: 1519, Message: "UUID not found"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNetworkError(tt.err)
			if result != tt.expected {
				t.Errorf("IsNetworkError() = %v, want %v for error: %v", result, tt.expected, tt.err)
			}
		})
	}
}
