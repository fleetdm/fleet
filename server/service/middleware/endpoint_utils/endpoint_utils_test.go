package endpoint_utils

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractIP_SkipsLocalhost(t *testing.T) {
	tests := []struct {
		name        string
		remoteAddr  string
		headers     map[string][]string
		expectedIP  string
		description string
	}{
		{
			name:       "X-Forwarded-For with localhost first, real IP second",
			remoteAddr: "192.168.1.100:8080",
			headers: map[string][]string{
				"X-Forwarded-For": {"127.0.0.1, 203.0.113.45"},
			},
			expectedIP:  "203.0.113.45",
			description: "Should skip localhost and return the first non-localhost IP",
		},
		{
			name:       "X-Forwarded-For with IPv6 localhost first, real IP second",
			remoteAddr: "192.168.1.100:8080",
			headers: map[string][]string{
				"X-Forwarded-For": {"::1, 203.0.113.45"},
			},
			expectedIP:  "203.0.113.45",
			description: "Should skip IPv6 localhost and return the first non-localhost IP",
		},
		{
			name:       "X-Forwarded-For with multiple localhost IPs and one real IP",
			remoteAddr: "192.168.1.100:8080",
			headers: map[string][]string{
				"X-Forwarded-For": {"127.0.0.1, ::1, 203.0.113.45, 198.51.100.22"},
			},
			expectedIP:  "203.0.113.45",
			description: "Should skip all localhost IPs and return the first non-localhost IP",
		},
		{
			name:       "X-Forwarded-For with only localhost IPs",
			remoteAddr: "192.168.1.100:8080",
			headers: map[string][]string{
				"X-Forwarded-For": {"127.0.0.1, ::1"},
			},
			expectedIP:  "127.0.0.1",
			description: "Should fallback to first IP when all are localhost",
		},
		{
			name:       "X-Forwarded-For with only IPv4 localhost",
			remoteAddr: "192.168.1.100:8080",
			headers: map[string][]string{
				"X-Forwarded-For": {"127.0.0.1"},
			},
			expectedIP:  "127.0.0.1",
			description: "Should fallback to localhost when it's the only IP",
		},
		{
			name:       "X-Forwarded-For with only IPv6 localhost",
			remoteAddr: "192.168.1.100:8080",
			headers: map[string][]string{
				"X-Forwarded-For": {"::1"},
			},
			expectedIP:  "::1",
			description: "Should fallback to IPv6 localhost when it's the only IP",
		},
		{
			name:       "Multiple X-Forwarded-For headers with localhost in first, real IP in second",
			remoteAddr: "192.168.1.100:8080",
			headers: map[string][]string{
				"X-Forwarded-For": {"127.0.0.1", "203.0.113.45"},
			},
			expectedIP:  "203.0.113.45",
			description: "Should check multiple X-Forwarded-For headers and skip localhost",
		},
		{
			name:       "X-Forwarded-For with whitespace around IPs",
			remoteAddr: "192.168.1.100:8080",
			headers: map[string][]string{
				"X-Forwarded-For": {" 127.0.0.1 ,  203.0.113.45  "},
			},
			expectedIP:  "203.0.113.45",
			description: "Should handle whitespace around IPs and skip localhost",
		},
		{
			name:       "True-Client-IP with localhost (should use localhost directly)",
			remoteAddr: "192.168.1.100:8080",
			headers: map[string][]string{
				"True-Client-IP": {"127.0.0.1"},
			},
			expectedIP:  "127.0.0.1",
			description: "True-Client-IP header is used directly without localhost checking",
		},
		{
			name:       "X-Real-IP with localhost (should use localhost directly)",
			remoteAddr: "192.168.1.100:8080",
			headers: map[string][]string{
				"X-Real-IP": {"127.0.0.1"},
			},
			expectedIP:  "127.0.0.1",
			description: "X-Real-IP header is used directly without localhost checking",
		},
		{
			name:        "No headers, just RemoteAddr",
			remoteAddr:  "203.0.113.45:8080",
			headers:     map[string][]string{},
			expectedIP:  "203.0.113.45",
			description: "Should extract IP from RemoteAddr when no headers present",
		},
		{
			name:       "Real IP first in X-Forwarded-For",
			remoteAddr: "192.168.1.100:8080",
			headers: map[string][]string{
				"X-Forwarded-For": {"203.0.113.45, 127.0.0.1"},
			},
			expectedIP:  "203.0.113.45",
			description: "Should return first IP when it's not localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				RemoteAddr: tt.remoteAddr,
				Header:     make(http.Header),
			}

			// Set headers
			for key, values := range tt.headers {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}

			result := ExtractIP(req)
			assert.Equal(t, tt.expectedIP, result, tt.description)
		})
	}
}
