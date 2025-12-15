package main

import (
	"context"
	"strings"
	"testing"
)

func TestParseMDATPHealthOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name: "Basic health output",
			input: `Healthy                                     : true
Licensed                                    : true
App version                                 : 101.23.45
Engine version                              : 1.1.19600.2`,
			expected: map[string]string{
				"healthy":        "true",
				"licensed":       "true",
				"app_version":    "101.23.45",
				"engine_version": "1.1.19600.2",
			},
		},
		{
			name: "Values with quotes",
			input: `Healthy                                     : "true"
Licensed                                    : "false"
Release ring                                : "beta"`,
			expected: map[string]string{
				"healthy":      "true",
				"licensed":     "false",
				"release_ring": "beta",
			},
		},
		{
			name: "Multi-word keys with spaces",
			input: `Real Time Protection Enabled                : true
Behavior Monitoring                         : enabled
Tamper Protection                           : managed`,
			expected: map[string]string{
				"real_time_protection_enabled": "true",
				"behavior_monitoring":          "enabled",
				"tamper_protection":            "managed",
			},
		},
		{
			name: "Ignores empty lines and ATTENTION",
			input: `Healthy                                     : true

ATTENTION: Some warning message here

Licensed                                    : true`,
			expected: map[string]string{
				"healthy":  "true",
				"licensed": "true",
			},
		},
		{
			name: "Complex values with special characters",
			input: `Machine GUID                                : "12345-abcde-67890-fghij"
Definitions Version                         : "1.387.123"
Conflicting Applications                    : "Office 365, Slack"`,
			expected: map[string]string{
				"machine_guid":             "12345-abcde-67890-fghij",
				"definitions_version":      "1.387.123",
				"conflicting_applications": "Office 365, Slack",
			},
		},
		{
			name:     "Empty input",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:     "Only whitespace and empty lines",
			input:    "\n\n   \n\t\n",
			expected: map[string]string{},
		},
		{
			name: "Mixed case keys",
			input: `HEALTHY                                     : true
Licensed                                    : true
Engine Version                              : 1.1.19600.2`,
			expected: map[string]string{
				"healthy":        "true",
				"licensed":       "true",
				"engine_version": "1.1.19600.2",
			},
		},
		{
			name: "Values with leading/trailing whitespace",
			input: `Healthy                                     :    true   
Licensed                                    :   false  `,
			expected: map[string]string{
				"healthy":  "true",
				"licensed": "false",
			},
		},
		{
			name: "Malformed lines are skipped",
			input: `Healthy                                     : true
This line has no colon
Licensed                                    : false
Another bad line`,
			expected: map[string]string{
				"healthy":  "true",
				"licensed": "false",
			},
		},
		{
			name: "Values with colons",
			input: `Last Updated                                : 2024-01-15 10:30:45
Time Zone                                   : UTC-05:00`,
			expected: map[string]string{
				"last_updated": "2024-01-15 10:30:45",
				"time_zone":    "UTC-05:00",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMDATPHealthOutput(tt.input)

			// Check all expected keys are present
			for key, expectedValue := range tt.expected {
				value, exists := result[key]
				if !exists {
					t.Errorf("Expected key %q not found in result", key)
					continue
				}
				if value != expectedValue {
					t.Errorf("For key %q: expected %q, got %q", key, expectedValue, value)
				}
			}

			// Check no unexpected keys are present
			for key := range result {
				if _, exists := tt.expected[key]; !exists {
					t.Errorf("Unexpected key %q in result with value %q", key, result[key])
				}
			}
		})
	}
}

func TestMdatpStatusColumns(t *testing.T) {
	columns := mdatpStatusColumns()

	if len(columns) == 0 {
		t.Error("Expected non-empty columns list")
	}

	expectedColumnNames := []string{
		"healthy",
		"health_issues",
		"licensed",
		"engine_version",
		"app_version",
		"org_id",
		"error",
	}

	columnNameMap := make(map[string]bool)
	for _, col := range columns {
		columnNameMap[col.Name] = true
	}

	for _, expected := range expectedColumnNames {
		if !columnNameMap[expected] {
			t.Errorf("Expected column %q not found", expected)
		}
	}

	// Verify all columns are TextColumn type
	for _, col := range columns {
		if col.Type != "TEXT" {
			t.Errorf("Column %q has unexpected type %q, expected TEXT", col.Name, col.Type)
		}
	}
}

func TestHealthOutputRegexPattern(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldMatch   bool
		expectedKey   string
		expectedValue string
	}{
		{
			name:          "Standard format",
			input:         "Healthy                                     : true",
			shouldMatch:   true,
			expectedKey:   "Healthy",
			expectedValue: "true",
		},
		{
			name:          "Minimal spacing",
			input:         "Key:value",
			shouldMatch:   true,
			expectedKey:   "Key",
			expectedValue: "value",
		},
		{
			name:          "Extra spaces around colon",
			input:         "Key   :   value",
			shouldMatch:   true,
			expectedKey:   "Key",
			expectedValue: "value",
		},
		{
			name:          "Quoted value",
			input:         `Value with spaces                           : "hello world"`,
			shouldMatch:   true,
			expectedKey:   "Value with spaces",
			expectedValue: `"hello world"`,
		},
		{
			name:        "No colon",
			input:       "This line has no separator",
			shouldMatch: false,
		},
		{
			name:        "Empty line",
			input:       "",
			shouldMatch: false,
		},
		{
			name:          "Numeric value",
			input:         "Count                                       : 42",
			shouldMatch:   true,
			expectedKey:   "Count",
			expectedValue: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := healthOutputRegex.FindStringSubmatch(tt.input)
			if tt.shouldMatch {
				if len(matches) != 3 {
					t.Errorf("Expected match with 3 groups, got %d", len(matches))
					return
				}
				if strings.TrimSpace(matches[1]) != tt.expectedKey {
					t.Errorf("Expected key %q, got %q", tt.expectedKey, strings.TrimSpace(matches[1]))
				}
				if strings.TrimSpace(matches[2]) != tt.expectedValue {
					t.Errorf("Expected value %q, got %q", tt.expectedValue, strings.TrimSpace(matches[2]))
				}
			} else {
				if len(matches) > 0 {
					t.Errorf("Expected no match but got: %v", matches)
				}
			}
		})
	}
}

func TestGenerateMDATPStatusWithoutBinary(t *testing.T) {
	// This test ensures generateMDATPStatus handles the case where mdatp binary is not found
	result, err := generateMDATPStatus(context.Background(), nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 result row, got %d", len(result))
	}

	// Should contain an error field since the binary won't be found
	if row := result[0]; row["error"] == "" {
		t.Error("Expected error field to be populated when mdatp binary is not found")
	}
}

func BenchmarkParseMDATPHealthOutput(b *testing.B) {
	input := `Healthy                                     : true
Licensed                                    : true
Engine version                              : 1.1.19600.2
App version                                 : 101.23.45
Org Id                                      : 00000000-0000-0000-0000-000000000000
Log level                                   : "warning"
Machine GUID                                : "12345-67890"
Release ring                                : "beta"
Product expiration                          : "2025-12-31"
Cloud enabled                               : true
Passive mode enabled                        : false
Behavior monitoring                         : enabled
Real Time Protection Enabled                : true
Real Time Protection Available              : true
Tamper Protection                           : managed
Automatic definition update enabled         : true
Definitions updated                         : "2024-01-15 10:30:45"
Definitions version                         : "1.387.123"
EDR Device Tags                             : "tag1,tag2,tag3"
Network Protection Status                   : enabled
Data Loss Prevention Status                 : enabled
Full Disk Access Enabled                    : true
Troubleshooting mode                        : disabled`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseMDATPHealthOutput(input)
	}
}
