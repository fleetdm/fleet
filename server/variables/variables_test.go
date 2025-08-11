package variables

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFind(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]struct{}
	}{
		{
			name:     "no variables",
			content:  "This is a plain text without any variables",
			expected: nil,
		},
		{
			name:    "single variable without braces",
			content: "Device ID: $FLEET_VAR_HOST_UUID",
			expected: map[string]struct{}{
				"HOST_UUID": {},
			},
		},
		{
			name:    "single variable with braces",
			content: "Device ID: ${FLEET_VAR_HOST_UUID}",
			expected: map[string]struct{}{
				"HOST_UUID": {},
			},
		},
		{
			name:    "multiple different variables",
			content: "Host: $FLEET_VAR_HOST_UUID, Email: ${FLEET_VAR_HOST_EMAIL}, Serial: $FLEET_VAR_HOST_SERIAL",
			expected: map[string]struct{}{
				"HOST_UUID":   {},
				"HOST_EMAIL":  {},
				"HOST_SERIAL": {},
			},
		},
		{
			name:    "duplicate variables",
			content: "ID1: $FLEET_VAR_HOST_UUID, ID2: ${FLEET_VAR_HOST_UUID}, ID3: $FLEET_VAR_HOST_UUID",
			expected: map[string]struct{}{
				"HOST_UUID": {},
			},
		},
		{
			name:    "variables in XML content",
			content: `<Replace><Data>Device: $FLEET_VAR_HOST_UUID</Data></Replace>`,
			expected: map[string]struct{}{
				"HOST_UUID": {},
			},
		},
		{
			name:    "mixed case sensitivity",
			content: "Valid: $FLEET_VAR_HOST_UUID, Invalid: $fleet_var_host_uuid",
			expected: map[string]struct{}{
				"HOST_UUID": {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Find(tt.content)
			assert.Equal(t, tt.expected, result)

			// Also verify that Contains returns the expected value
			// If Find returns nil (no variables), Contains should return false
			// If Find returns a non-nil map, Contains should return true
			expectedContains := tt.expected != nil
			assert.Equal(t, expectedContains, Contains(tt.content),
				"Contains() should return %v when Find() returns %v", expectedContains, tt.expected)
		})
	}
}

func TestFindKeepDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "no variables",
			content:  "Plain text",
			expected: nil,
		},
		{
			name:     "single occurrence",
			content:  "ID: $FLEET_VAR_HOST_UUID",
			expected: []string{"HOST_UUID"},
		},
		{
			name:     "duplicate occurrences",
			content:  "ID1: $FLEET_VAR_HOST_UUID, ID2: ${FLEET_VAR_HOST_UUID}, ID3: $FLEET_VAR_HOST_UUID",
			expected: []string{"HOST_UUID", "HOST_UUID", "HOST_UUID"},
		},
		{
			name:     "mixed variables with duplicates",
			content:  "$FLEET_VAR_HOST_UUID, $FLEET_VAR_HOST_EMAIL, ${FLEET_VAR_HOST_UUID}",
			expected: []string{"HOST_UUID", "HOST_EMAIL", "HOST_UUID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindKeepDuplicates(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplace(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		variableName string
		value        string
		expected     string
	}{
		{
			name:         "replace variable without braces",
			content:      "Device ID: $FLEET_VAR_HOST_UUID",
			variableName: "HOST_UUID",
			value:        "123-456-789",
			expected:     "Device ID: 123-456-789",
		},
		{
			name:         "replace variable with braces",
			content:      "Device ID: ${FLEET_VAR_HOST_UUID}",
			variableName: "HOST_UUID",
			value:        "123-456-789",
			expected:     "Device ID: 123-456-789",
		},
		{
			name:         "replace both formats",
			content:      "ID1: $FLEET_VAR_HOST_UUID, ID2: ${FLEET_VAR_HOST_UUID}",
			variableName: "HOST_UUID",
			value:        "abc-def",
			expected:     "ID1: abc-def, ID2: abc-def",
		},
		{
			name:         "no replacement when variable not present",
			content:      "Email: $FLEET_VAR_HOST_EMAIL",
			variableName: "HOST_UUID",
			value:        "123",
			expected:     "Email: $FLEET_VAR_HOST_EMAIL",
		},
		{
			name:         "replace with empty value",
			content:      "ID: $FLEET_VAR_HOST_UUID",
			variableName: "HOST_UUID",
			value:        "",
			expected:     "ID: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Replace(tt.content, tt.variableName, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}
