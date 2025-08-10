package variables

import (
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name:    "variables in XML",
			content: `<Replace><Data>ID: $FLEET_VAR_HOST_UUID</Data><Email>${FLEET_VAR_HOST_EMAIL}</Email></Replace>`,
			expected: map[string]struct{}{
				"HOST_UUID":  {},
				"HOST_EMAIL": {},
			},
		},
		{
			name:    "variables with underscores in name",
			content: "$FLEET_VAR_NDES_SCEP_CHALLENGE and ${FLEET_VAR_HOST_END_USER_EMAIL_IDP}",
			expected: map[string]struct{}{
				"NDES_SCEP_CHALLENGE":     {},
				"HOST_END_USER_EMAIL_IDP": {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Find(tt.content)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
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
			content:  "$FLEET_VAR_HOST_UUID",
			expected: []string{"HOST_UUID"},
		},
		{
			name:     "multiple occurrences of same variable",
			content:  "First: $FLEET_VAR_HOST_UUID, Second: ${FLEET_VAR_HOST_UUID}, Third: $FLEET_VAR_HOST_UUID",
			expected: []string{"HOST_UUID", "HOST_UUID", "HOST_UUID"},
		},
		{
			name:     "mixed variables with duplicates",
			content:  "$FLEET_VAR_HOST_UUID ${FLEET_VAR_HOST_EMAIL} $FLEET_VAR_HOST_UUID",
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

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "no variables",
			content:  "Plain text without variables",
			expected: false,
		},
		{
			name:     "contains variable without braces",
			content:  "Device: $FLEET_VAR_HOST_UUID",
			expected: true,
		},
		{
			name:     "contains variable with braces",
			content:  "Device: ${FLEET_VAR_HOST_UUID}",
			expected: true,
		},
		{
			name:     "partial variable name",
			content:  "$FLEET_VAR_ or FLEET_VAR_HOST",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsSpecific(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		variableName string
		expected     bool
	}{
		{
			name:         "contains specific variable without braces",
			content:      "ID: $FLEET_VAR_HOST_UUID",
			variableName: "HOST_UUID",
			expected:     true,
		},
		{
			name:         "contains specific variable with braces",
			content:      "ID: ${FLEET_VAR_HOST_UUID}",
			variableName: "HOST_UUID",
			expected:     true,
		},
		{
			name:         "does not contain specific variable",
			content:      "Email: $FLEET_VAR_HOST_EMAIL",
			variableName: "HOST_UUID",
			expected:     false,
		},
		{
			name:         "contains both formats",
			content:      "$FLEET_VAR_HOST_UUID and ${FLEET_VAR_HOST_UUID}",
			variableName: "HOST_UUID",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsSpecific(tt.content, tt.variableName)
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

func TestReplaceAll(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		values   map[string]string
		expected string
	}{
		{
			name:    "replace multiple variables",
			content: "Host: $FLEET_VAR_HOST_UUID, Email: ${FLEET_VAR_HOST_EMAIL}",
			values: map[string]string{
				"HOST_UUID":  "123-456",
				"HOST_EMAIL": "test@example.com",
			},
			expected: "Host: 123-456, Email: test@example.com",
		},
		{
			name:    "partial replacement",
			content: "Host: $FLEET_VAR_HOST_UUID, Email: ${FLEET_VAR_HOST_EMAIL}, Serial: $FLEET_VAR_HOST_SERIAL",
			values: map[string]string{
				"HOST_UUID": "123-456",
			},
			expected: "Host: 123-456, Email: ${FLEET_VAR_HOST_EMAIL}, Serial: $FLEET_VAR_HOST_SERIAL",
		},
		{
			name:     "empty values map",
			content:  "Host: $FLEET_VAR_HOST_UUID",
			values:   map[string]string{},
			expected: "Host: $FLEET_VAR_HOST_UUID",
		},
		{
			name:     "nil values map",
			content:  "Host: $FLEET_VAR_HOST_UUID",
			values:   nil,
			expected: "Host: $FLEET_VAR_HOST_UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplaceAll(tt.content, tt.values)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		validVariables  map[string]struct{}
		expectedInvalid []string
	}{
		{
			name:    "all variables valid",
			content: "Host: $FLEET_VAR_HOST_UUID, Email: ${FLEET_VAR_HOST_EMAIL}",
			validVariables: map[string]struct{}{
				"HOST_UUID":  {},
				"HOST_EMAIL": {},
			},
			expectedInvalid: nil,
		},
		{
			name:    "some variables invalid",
			content: "Host: $FLEET_VAR_HOST_UUID, Invalid: ${FLEET_VAR_INVALID_VAR}",
			validVariables: map[string]struct{}{
				"HOST_UUID": {},
			},
			expectedInvalid: []string{"INVALID_VAR"},
		},
		{
			name:            "all variables invalid",
			content:         "Invalid1: $FLEET_VAR_INVALID1, Invalid2: ${FLEET_VAR_INVALID2}",
			validVariables:  map[string]struct{}{},
			expectedInvalid: []string{"INVALID1", "INVALID2"},
		},
		{
			name:            "no variables in content",
			content:         "Plain text without variables",
			validVariables:  map[string]struct{}{"HOST_UUID": {}},
			expectedInvalid: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Validate(tt.content, tt.validVariables)

			// Sort both slices for comparison since map iteration order is not guaranteed
			if result != nil && tt.expectedInvalid != nil {
				sort.Strings(result)
				sort.Strings(tt.expectedInvalid)
			}

			if tt.expectedInvalid == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expectedInvalid, result)
			}
		})
	}
}

func TestExtractVariableName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "variable without braces",
			input:    "$FLEET_VAR_HOST_UUID",
			expected: "HOST_UUID",
		},
		{
			name:     "variable with braces",
			input:    "${FLEET_VAR_HOST_UUID}",
			expected: "HOST_UUID",
		},
		{
			name:     "variable with whitespace",
			input:    "  $FLEET_VAR_HOST_UUID  ",
			expected: "HOST_UUID",
		},
		{
			name:     "not a variable",
			input:    "plain text",
			expected: "",
		},
		{
			name:     "partial variable",
			input:    "FLEET_VAR_HOST_UUID",
			expected: "",
		},
		{
			name:     "malformed braces",
			input:    "${FLEET_VAR_HOST_UUID",
			expected: "",
		},
		{
			name:     "variable with underscores",
			input:    "$FLEET_VAR_HOST_END_USER_EMAIL",
			expected: "HOST_END_USER_EMAIL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractVariableName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatVariable(t *testing.T) {
	tests := []struct {
		name         string
		variableName string
		useBraces    bool
		expected     string
	}{
		{
			name:         "format without braces",
			variableName: "HOST_UUID",
			useBraces:    false,
			expected:     "$FLEET_VAR_HOST_UUID",
		},
		{
			name:         "format with braces",
			variableName: "HOST_UUID",
			useBraces:    true,
			expected:     "${FLEET_VAR_HOST_UUID}",
		},
		{
			name:         "format with underscores without braces",
			variableName: "HOST_END_USER_EMAIL",
			useBraces:    false,
			expected:     "$FLEET_VAR_HOST_END_USER_EMAIL",
		},
		{
			name:         "format with underscores with braces",
			variableName: "HOST_END_USER_EMAIL",
			useBraces:    true,
			expected:     "${FLEET_VAR_HOST_END_USER_EMAIL}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatVariable(tt.variableName, tt.useBraces)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFleetVariableRegex(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected [][]string
	}{
		{
			name:    "match without braces",
			content: "$FLEET_VAR_HOST_UUID",
			expected: [][]string{
				{"$FLEET_VAR_HOST_UUID", "$FLEET_VAR_HOST_UUID", "HOST_UUID", "", ""},
			},
		},
		{
			name:    "match with braces",
			content: "${FLEET_VAR_HOST_UUID}",
			expected: [][]string{
				{"${FLEET_VAR_HOST_UUID}", "", "", "${FLEET_VAR_HOST_UUID}", "HOST_UUID"},
			},
		},
		{
			name:    "multiple matches",
			content: "$FLEET_VAR_HOST_UUID and ${FLEET_VAR_HOST_EMAIL}",
			expected: [][]string{
				{"$FLEET_VAR_HOST_UUID", "$FLEET_VAR_HOST_UUID", "HOST_UUID", "", ""},
				{"${FLEET_VAR_HOST_EMAIL}", "", "", "${FLEET_VAR_HOST_EMAIL}", "HOST_EMAIL"},
			},
		},
		{
			name:     "no matches",
			content:  "plain text without variables",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := FleetVariableRegex.FindAllStringSubmatch(tt.content, -1)
			if !reflect.DeepEqual(matches, tt.expected) {
				t.Errorf("FleetVariableRegex.FindAllStringSubmatch() = %v, want %v", matches, tt.expected)
			}
		})
	}
}

func TestProfileDataVariableRegex(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected [][]string
	}{
		{
			name:    "match DigiCert data variable without braces",
			content: "$FLEET_VAR_DIGICERT_DATA_CERT",
			expected: [][]string{
				{"$FLEET_VAR_DIGICERT_DATA_CERT", "$FLEET_VAR_DIGICERT_DATA_CERT", "CERT", "", ""},
			},
		},
		{
			name:    "match DigiCert data variable with braces",
			content: "${FLEET_VAR_DIGICERT_DATA_KEY}",
			expected: [][]string{
				{"${FLEET_VAR_DIGICERT_DATA_KEY}", "", "", "${FLEET_VAR_DIGICERT_DATA_KEY}", "KEY"},
			},
		},
		{
			name:     "no match for regular Fleet variable",
			content:  "$FLEET_VAR_HOST_UUID",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := ProfileDataVariableRegex.FindAllStringSubmatch(tt.content, -1)
			require.Equal(t, tt.expected, matches)
		})
	}
}
