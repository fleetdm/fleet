package fleet

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoftwareIterQueryOptionsIsValid(t *testing.T) {
	testCases := []struct {
		excluded   []string
		included   []string
		isNotValid bool
	}{
		{
			excluded: nil,
			included: nil,
		},
		{
			excluded: []string{"a", "b"},
			included: nil,
		},
		{
			excluded: nil,
			included: []string{"a", "b"},
		},
		{
			excluded:   []string{"a", "b"},
			included:   []string{"a"},
			isNotValid: true,
		},
		{
			excluded:   []string{"a"},
			included:   []string{"a", "b"},
			isNotValid: true,
		},
		{
			excluded:   []string{"c"},
			included:   []string{"a", "b"},
			isNotValid: true,
		},
	}

	for _, tC := range testCases {
		sut := SoftwareIterQueryOptions{
			ExcludedSources: tC.excluded,
			IncludedSources: tC.included,
		}

		if tC.isNotValid {
			require.False(t, sut.IsValid())
		} else {
			require.True(t, sut.IsValid())
		}
	}
}

func TestParseSoftwareLastOpenedAtRowValue(t *testing.T) {
	// Some macOS apps return last_opened_at=-1.0 on apps
	// that were never opened.
	lastOpenedAt, err := ParseSoftwareLastOpenedAtRowValue("-1.0")
	require.NoError(t, err)
	require.Zero(t, lastOpenedAt)

	// Our software queries hardcode to 0 if such info is not available.
	lastOpenedAt, err = ParseSoftwareLastOpenedAtRowValue("0")
	require.NoError(t, err)
	require.Zero(t, lastOpenedAt)

	lastOpenedAt, err = ParseSoftwareLastOpenedAtRowValue("foobar")
	require.Error(t, err)
	require.Zero(t, lastOpenedAt)

	lastOpenedAt, err = ParseSoftwareLastOpenedAtRowValue("1694026958")
	require.NoError(t, err)
	require.NotZero(t, lastOpenedAt)
}

func TestEnhanceOutputDetails(t *testing.T) {
	tests := []struct {
		name                            string
		initial                         HostSoftwareInstallerResult
		expectedPreInstallQueryOutput   *string
		expectedOutput                  *string
		expectedPostInstallScriptOutput *string
	}{
		{
			name: "pending status",
			initial: HostSoftwareInstallerResult{
				Status: SoftwareInstallPending,
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  nil,
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with empty PreInstallQueryOutput",
			initial: HostSoftwareInstallerResult{
				Status:                SoftwareInstalled,
				PreInstallQueryOutput: ptr.String(""),
			},
			expectedPreInstallQueryOutput:   ptr.String(SoftwareInstallerQueryFailCopy),
			expectedOutput:                  nil,
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with non-empty PreInstallQueryOutput",
			initial: HostSoftwareInstallerResult{
				Status:                SoftwareInstalled,
				PreInstallQueryOutput: ptr.String("Some output"),
			},
			expectedPreInstallQueryOutput:   ptr.String(SoftwareInstallerQuerySuccessCopy),
			expectedOutput:                  nil,
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with nil PreInstallQueryOutput",
			initial: HostSoftwareInstallerResult{
				Status: SoftwareInstalled,
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  nil,
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with install scripts disabled",
			initial: HostSoftwareInstallerResult{
				Status:                SoftwareInstalled,
				InstallScriptExitCode: ptr.Int(-2),
				Output:                ptr.String(""),
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  ptr.String(SoftwareInstallerScriptsDisabledCopy),
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with failed install script",
			initial: HostSoftwareInstallerResult{
				Status:                SoftwareInstallFailed,
				InstallScriptExitCode: ptr.Int(1),
				Output:                ptr.String("Some install output"),
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  ptr.String(fmt.Sprintf(SoftwareInstallerInstallFailCopy, "Some install output")),
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with successful install script",
			initial: HostSoftwareInstallerResult{
				Status:                SoftwareInstalled,
				InstallScriptExitCode: ptr.Int(0),
				Output:                ptr.String("Some install output"),
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  ptr.String(fmt.Sprintf(SoftwareInstallerInstallSuccessCopy, "Some install output")),
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with successful post install script",
			initial: HostSoftwareInstallerResult{
				Status:                    SoftwareInstalled,
				InstallScriptExitCode:     ptr.Int(0),
				Output:                    ptr.String("Some install output"),
				PostInstallScriptExitCode: ptr.Int(0),
				PostInstallScriptOutput:   ptr.String("Some post install output"),
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  ptr.String(fmt.Sprintf(SoftwareInstallerInstallSuccessCopy, "Some install output")),
			expectedPostInstallScriptOutput: ptr.String(fmt.Sprintf(SoftwareInstallerPostInstallSuccessCopy, "Some post install output")),
		},
		{
			name: "non-pending status with failed post install script",
			initial: HostSoftwareInstallerResult{
				Status:                    SoftwareInstalled,
				InstallScriptExitCode:     ptr.Int(0),
				Output:                    ptr.String("Some install output"),
				PostInstallScriptExitCode: ptr.Int(1),
				PostInstallScriptOutput:   ptr.String("Some post install output"),
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  ptr.String(fmt.Sprintf(SoftwareInstallerInstallSuccessCopy, "Some install output")),
			expectedPostInstallScriptOutput: ptr.String(fmt.Sprintf(SoftwareInstallerPostInstallFailCopy, 1, "Some post install output")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.EnhanceOutputDetails()
			require.Equal(t, tt.expectedPreInstallQueryOutput, tt.initial.PreInstallQueryOutput)
			require.Equal(t, tt.expectedOutput, tt.initial.Output)
			require.Equal(t, tt.expectedPostInstallScriptOutput, tt.initial.PostInstallScriptOutput)
		})
	}
}

func TestHostSoftwareEntryMarshalJSON(t *testing.T) {
	// Test that HostSoftwareEntry properly marshals all fields including
	// InstalledPaths and PathSignatureInformation from the embedded Software struct
	hashValue := "abc123"
	entry := HostSoftwareEntry{
		Software: Software{
			ID:               1,
			Name:             "Test Software",
			Version:          "1.0.0",
			Source:           "chrome_extensions",
			BundleIdentifier: "com.test.software",
			ExtensionID:      "test-extension-id",
			ExtensionFor:     "chrome",
			Browser:          "",
			Release:          "1",
			Vendor:           "Test Vendor",
			Arch:             "x86_64",
			GenerateCPE:      "cpe:2.3:a:test:software:1.0.0:*:*:*:*:*:*:*",
			Vulnerabilities:  Vulnerabilities{},
			HostsCount:       5,
			ApplicationID:    ptr.String("com.test.app"),
		},
		InstalledPaths: []string{"/usr/local/bin/test", "/opt/test"},
		PathSignatureInformation: []PathSignatureInformation{
			{
				InstalledPath:  "/usr/local/bin/test",
				TeamIdentifier: "ABCDE12345",
				HashSha256:     &hashValue,
			},
		},
	}

	// Marshal to JSON
	data, err := entry.MarshalJSON()
	require.NoError(t, err)

	// Expected JSON with all fields including browser and extension_for
	expectedJSON := `{
		"id": 1,
		"name": "Test Software",
		"version": "1.0.0",
		"bundle_identifier": "com.test.software",
		"source": "chrome_extensions",
		"extension_id": "test-extension-id",
		"extension_for": "chrome",
		"display_name": "",
		"browser": "chrome",
		"release": "1",
		"vendor": "Test Vendor",
		"arch": "x86_64",
		"generated_cpe": "cpe:2.3:a:test:software:1.0.0:*:*:*:*:*:*:*",
		"vulnerabilities": [],
		"hosts_count": 5,
		"application_id": "com.test.app",
		"installed_paths": ["/usr/local/bin/test", "/opt/test"],
		"signature_information": [
			{
				"installed_path": "/usr/local/bin/test",
				"team_identifier": "ABCDE12345",
				"hash_sha256": "abc123"
			}
		]
	}`

	assert.JSONEq(t, expectedJSON, string(data))
}

func TestSoftwareMarshalJSONLastOpenedAt(t *testing.T) {
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		software      Software
		expectField   bool
		expectedValue any
		description   string
	}{
		{
			name: "supported source with nil last_opened_at",
			software: Software{
				Source:       "apps",
				LastOpenedAt: nil,
			},
			expectField:   true,
			expectedValue: "",
			description:   "Should return empty string for supported source with nil",
		},
		{
			name: "supported source with timestamp",
			software: Software{
				Source:       "programs",
				LastOpenedAt: &now,
			},
			expectField:   true,
			expectedValue: now.Format(time.RFC3339),
			description:   "Should return timestamp for supported source with value",
		},
		{
			name: "unsupported source with nil last_opened_at",
			software: Software{
				Source:       "chrome_extensions",
				LastOpenedAt: nil,
			},
			expectField:   false,
			expectedValue: nil,
			description:   "Should omit field for unsupported source",
		},
		{
			name: "unsupported source with timestamp",
			software: Software{
				Source:       "python_packages",
				LastOpenedAt: &now,
			},
			expectField:   false,
			expectedValue: nil,
			description:   "Should omit field for unsupported source even with value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.software.MarshalJSON()
			require.NoError(t, err)

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			if tt.expectField {
				require.Contains(t, result, "last_opened_at", tt.description)
				if tt.expectedValue == "" {
					assert.Equal(t, "", result["last_opened_at"], tt.description)
				} else {
					// For timestamps, check that it's a valid RFC3339 string
					timeStr, ok := result["last_opened_at"].(string)
					require.True(t, ok, "last_opened_at should be a string")
					parsedTime, err := time.Parse(time.RFC3339, timeStr)
					require.NoError(t, err)
					assert.True(t, parsedTime.Equal(now), tt.description)
				}
			} else {
				assert.NotContains(t, result, "last_opened_at", tt.description)
			}
		})
	}
}

func TestHostSoftwareEntryMarshalJSONLastOpenedAt(t *testing.T) {
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		entry         HostSoftwareEntry
		expectField   bool
		expectedValue any
		description   string
	}{
		{
			name: "supported source with nil last_opened_at",
			entry: HostSoftwareEntry{
				Software: Software{
					Source:       "deb_packages",
					LastOpenedAt: nil,
				},
			},
			expectField:   true,
			expectedValue: "",
			description:   "Should return empty string for supported source with nil",
		},
		{
			name: "supported source with timestamp",
			entry: HostSoftwareEntry{
				Software: Software{
					Source:       "rpm_packages",
					LastOpenedAt: &now,
				},
			},
			expectField:   true,
			expectedValue: now.Format(time.RFC3339),
			description:   "Should return timestamp for supported source with value",
		},
		{
			name: "unsupported source with nil last_opened_at",
			entry: HostSoftwareEntry{
				Software: Software{
					Source:       "chrome_extensions",
					LastOpenedAt: nil,
				},
			},
			expectField:   false,
			expectedValue: nil,
			description:   "Should omit field for unsupported source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.entry.MarshalJSON()
			require.NoError(t, err)

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			if tt.expectField {
				require.Contains(t, result, "last_opened_at", tt.description)
				if tt.expectedValue == "" {
					assert.Equal(t, "", result["last_opened_at"], tt.description)
				} else {
					// For timestamps, check that it's a valid RFC3339 string
					timeStr, ok := result["last_opened_at"].(string)
					require.True(t, ok, "last_opened_at should be a string")
					parsedTime, err := time.Parse(time.RFC3339, timeStr)
					require.NoError(t, err)
					assert.True(t, parsedTime.Equal(now), tt.description)
				}
			} else {
				assert.NotContains(t, result, "last_opened_at", tt.description)
			}
		})
	}
}

func TestHostSoftwareInstalledVersionMarshalJSONLastOpenedAt(t *testing.T) {
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		version       HostSoftwareInstalledVersion
		expectField   bool
		expectedValue any
		description   string
	}{
		{
			name: "supported source with nil last_opened_at",
			version: HostSoftwareInstalledVersion{
				Source:       "apps",
				LastOpenedAt: nil,
			},
			expectField:   true,
			expectedValue: "",
			description:   "Should return empty string for supported source with nil",
		},
		{
			name: "supported source with timestamp",
			version: HostSoftwareInstalledVersion{
				Source:       "programs",
				LastOpenedAt: &now,
			},
			expectField:   true,
			expectedValue: now.Format(time.RFC3339),
			description:   "Should return timestamp for supported source with value",
		},
		{
			name: "unsupported source with nil last_opened_at",
			version: HostSoftwareInstalledVersion{
				Source:       "chrome_extensions",
				LastOpenedAt: nil,
			},
			expectField:   false,
			expectedValue: nil,
			description:   "Should omit field for unsupported source",
		},
		{
			name: "unsupported source with timestamp",
			version: HostSoftwareInstalledVersion{
				Source:       "python_packages",
				LastOpenedAt: &now,
			},
			expectField:   false,
			expectedValue: nil,
			description:   "Should omit field for unsupported source even with value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.version.MarshalJSON()
			require.NoError(t, err)

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			if tt.expectField {
				require.Contains(t, result, "last_opened_at", tt.description)
				if tt.expectedValue == "" {
					assert.Equal(t, "", result["last_opened_at"], tt.description)
				} else {
					// For timestamps, check that it's a valid RFC3339 string
					timeStr, ok := result["last_opened_at"].(string)
					require.True(t, ok, "last_opened_at should be a string")
					parsedTime, err := time.Parse(time.RFC3339, timeStr)
					require.NoError(t, err)
					assert.True(t, parsedTime.Equal(now), tt.description)
				}
			} else {
				assert.NotContains(t, result, "last_opened_at", tt.description)
			}
		})
	}
}

func TestSoftwareUnmarshalJSON(t *testing.T) {
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	nowStr := now.Format(time.RFC3339)

	tests := []struct {
		name          string
		input         string
		expectedValue *time.Time
		expectError   bool
	}{
		{
			name:          "null value",
			input:         `{"id": 1, "name": "Test", "source": "apps", "last_opened_at": null}`,
			expectedValue: nil,
			expectError:   false,
		},
		{
			name:          "empty string",
			input:         `{"id": 1, "name": "Test", "source": "apps", "last_opened_at": ""}`,
			expectedValue: nil,
			expectError:   false,
		},
		{
			name:          "valid timestamp",
			input:         fmt.Sprintf(`{"id": 1, "name": "Test", "source": "apps", "last_opened_at": %q}`, nowStr),
			expectedValue: &now,
			expectError:   false,
		},
		{
			name:          "missing field",
			input:         `{"id": 1, "name": "Test", "source": "apps"}`,
			expectedValue: nil,
			expectError:   false,
		},
		{
			name:          "invalid timestamp format",
			input:         `{"id": 1, "name": "Test", "source": "apps", "last_opened_at": "invalid"}`,
			expectedValue: nil,
			expectError:   true,
		},
		{
			name:          "invalid JSON",
			input:         `{invalid json}`,
			expectedValue: nil,
			expectError:   true,
		},
		{
			name:          "boolean value (should error)",
			input:         `{"id": 1, "name": "Test", "source": "apps", "last_opened_at": true}`,
			expectedValue: nil,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Software
			err := json.Unmarshal([]byte(tt.input), &s)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expectedValue == nil {
					require.Nil(t, s.LastOpenedAt)
				} else {
					require.NotNil(t, s.LastOpenedAt)
					require.True(t, s.LastOpenedAt.Equal(*tt.expectedValue))
				}
			}
		})
	}

	// Test round-trip marshaling/unmarshaling
	t.Run("round-trip", func(t *testing.T) {
		original := Software{
			ID:           1,
			Name:         "Test Software",
			Source:       "apps",
			Version:      "1.0.0",
			LastOpenedAt: &now,
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var unmarshaled Software
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		require.NotNil(t, unmarshaled.LastOpenedAt)
		require.True(t, unmarshaled.LastOpenedAt.Equal(now))
	})
}

func TestHostSoftwareEntryUnmarshalJSON(t *testing.T) {
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	nowStr := now.Format(time.RFC3339)

	tests := []struct {
		name          string
		input         string
		expectedValue *time.Time
		expectError   bool
	}{
		{
			name:          "null value",
			input:         `{"id": 1, "name": "Test", "source": "apps", "last_opened_at": null, "installed_paths": []}`,
			expectedValue: nil,
			expectError:   false,
		},
		{
			name:          "empty string",
			input:         `{"id": 1, "name": "Test", "source": "apps", "last_opened_at": "", "installed_paths": []}`,
			expectedValue: nil,
			expectError:   false,
		},
		{
			name:          "valid timestamp",
			input:         fmt.Sprintf(`{"id": 1, "name": "Test", "version": "1.0.0", "source": "apps", "last_opened_at": %q, "installed_paths": []}`, nowStr),
			expectedValue: &now,
			expectError:   false,
		},
		{
			name:          "missing field",
			input:         `{"id": 1, "name": "Test", "source": "apps", "installed_paths": []}`,
			expectedValue: nil,
			expectError:   false,
		},
		{
			name:          "invalid timestamp format",
			input:         `{"id": 1, "name": "Test", "source": "apps", "last_opened_at": "invalid", "installed_paths": []}`,
			expectedValue: nil,
			expectError:   true,
		},
		{
			name:          "invalid JSON",
			input:         `{invalid json}`,
			expectedValue: nil,
			expectError:   true,
		},
		{
			name:          "with installed_paths and signature_information",
			input:         fmt.Sprintf(`{"id": 1, "name": "Test", "version": "1.0.0", "source": "apps", "last_opened_at": %q, "installed_paths": ["/usr/local/bin/test"], "signature_information": [{"installed_path": "/usr/local/bin/test", "team_identifier": "ABC123"}]}`, nowStr),
			expectedValue: &now,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hse HostSoftwareEntry
			err := json.Unmarshal([]byte(tt.input), &hse)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expectedValue == nil {
					require.Nil(t, hse.LastOpenedAt)
				} else {
					require.NotNil(t, hse.LastOpenedAt, "LastOpenedAt should be set for test case: %s", tt.name)
					require.True(t, hse.LastOpenedAt.Equal(*tt.expectedValue))
				}
			}
		})
	}

	// Test round-trip marshaling/unmarshaling
	t.Run("round-trip", func(t *testing.T) {
		original := HostSoftwareEntry{
			Software: Software{
				ID:           1,
				Name:         "Test Software",
				Source:       "apps",
				Version:      "1.0.0",
				LastOpenedAt: &now,
			},
			InstalledPaths: []string{"/usr/local/bin/test"},
			PathSignatureInformation: []PathSignatureInformation{
				{
					InstalledPath:  "/usr/local/bin/test",
					TeamIdentifier: "ABC123",
				},
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		// Verify the JSON includes last_opened_at for supported sources
		var jsonMap map[string]any
		err = json.Unmarshal(data, &jsonMap)
		require.NoError(t, err)
		require.Contains(t, jsonMap, "last_opened_at", "JSON should include last_opened_at for supported sources")

		var unmarshaled HostSoftwareEntry
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		// Verify all fields are preserved in round-trip
		require.Equal(t, original.Source, unmarshaled.Source, "Source should be preserved")
		require.NotNil(t, unmarshaled.LastOpenedAt, "LastOpenedAt should be preserved for supported sources")
		require.True(t, unmarshaled.LastOpenedAt.Equal(now))
		require.Equal(t, original.InstalledPaths, unmarshaled.InstalledPaths)
		require.Equal(t, original.PathSignatureInformation, unmarshaled.PathSignatureInformation)
	})
}

func TestHostSoftwareInstalledVersionUnmarshalJSON(t *testing.T) {
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	nowStr := now.Format(time.RFC3339)

	tests := []struct {
		name          string
		input         string
		expectedValue *time.Time
		expectError   bool
	}{
		{
			name:          "null value",
			input:         `{"version": "1.0.0", "bundle_identifier": "com.test", "source": "apps", "last_opened_at": null, "vulnerabilities": [], "installed_paths": []}`,
			expectedValue: nil,
			expectError:   false,
		},
		{
			name:          "empty string",
			input:         `{"version": "1.0.0", "bundle_identifier": "com.test", "source": "apps", "last_opened_at": "", "vulnerabilities": [], "installed_paths": []}`,
			expectedValue: nil,
			expectError:   false,
		},
		{
			name:          "valid timestamp",
			input:         fmt.Sprintf(`{"version": "1.0.0", "bundle_identifier": "com.test", "source": "apps", "last_opened_at": %q, "vulnerabilities": [], "installed_paths": []}`, nowStr),
			expectedValue: &now,
			expectError:   false,
		},
		{
			name:          "missing field",
			input:         `{"version": "1.0.0", "bundle_identifier": "com.test", "source": "apps", "vulnerabilities": [], "installed_paths": []}`,
			expectedValue: nil,
			expectError:   false,
		},
		{
			name:          "invalid timestamp format",
			input:         `{"version": "1.0.0", "bundle_identifier": "com.test", "source": "apps", "last_opened_at": "invalid", "vulnerabilities": [], "installed_paths": []}`,
			expectedValue: nil,
			expectError:   true,
		},
		{
			name:          "invalid JSON",
			input:         `{invalid json}`,
			expectedValue: nil,
			expectError:   true,
		},
		{
			name:          "with vulnerabilities and installed_paths",
			input:         fmt.Sprintf(`{"version": "1.0.0", "bundle_identifier": "com.test", "source": "apps", "last_opened_at": %q, "vulnerabilities": ["CVE-2023-1234"], "installed_paths": ["/usr/local/bin/test"], "signature_information": [{"installed_path": "/usr/local/bin/test"}]}`, nowStr),
			expectedValue: &now,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hsv HostSoftwareInstalledVersion
			err := json.Unmarshal([]byte(tt.input), &hsv)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expectedValue == nil {
					require.Nil(t, hsv.LastOpenedAt)
				} else {
					require.NotNil(t, hsv.LastOpenedAt)
					require.True(t, hsv.LastOpenedAt.Equal(*tt.expectedValue))
				}
			}
		})
	}

	// Test round-trip marshaling/unmarshaling
	t.Run("round-trip", func(t *testing.T) {
		original := HostSoftwareInstalledVersion{
			Version:          "1.0.0",
			BundleIdentifier: "com.test",
			Source:           "apps",
			LastOpenedAt:     &now,
			Vulnerabilities:  []string{"CVE-2023-1234"},
			InstalledPaths:   []string{"/usr/local/bin/test"},
			SignatureInformation: []PathSignatureInformation{
				{
					InstalledPath:  "/usr/local/bin/test",
					TeamIdentifier: "ABC123",
				},
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var unmarshaled HostSoftwareInstalledVersion
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		require.NotNil(t, unmarshaled.LastOpenedAt)
		require.True(t, unmarshaled.LastOpenedAt.Equal(now))
		require.Equal(t, original.Vulnerabilities, unmarshaled.Vulnerabilities)
		require.Equal(t, original.InstalledPaths, unmarshaled.InstalledPaths)
		require.Equal(t, original.SignatureInformation, unmarshaled.SignatureInformation)
	})
}

func TestAutoUpdateScheduleValidation(t *testing.T) {
	testCases := []struct {
		name     string
		schedule SoftwareAutoUpdateSchedule
		isValid  bool
	}{
		{
			name: "schedule disabled without times",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(false),
					AutoUpdateStartTime: nil,
					AutoUpdateEndTime:   nil,
				},
			},
			isValid: false,
		},
		{
			name: "schedule disabled with valid times",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(false),
					AutoUpdateStartTime: ptr.String("14:30"),
					AutoUpdateEndTime:   ptr.String("15:30"),
				},
			},
			isValid: true,
		},
		{
			name: "missing start time",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(true),
					AutoUpdateStartTime: nil,
					AutoUpdateEndTime:   ptr.String("15:30"),
				},
			},
			isValid: false,
		},
		{
			name: "missing end time",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(true),
					AutoUpdateStartTime: ptr.String("14:30"),
					AutoUpdateEndTime:   nil,
				},
			},
			isValid: false,
		},
		{
			name: "empty start time",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(true),
					AutoUpdateStartTime: ptr.String(""),
					AutoUpdateEndTime:   ptr.String("15:30"),
				},
			},
			isValid: false,
		},
		{
			name: "empty end time",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(true),
					AutoUpdateStartTime: ptr.String("14:30"),
					AutoUpdateEndTime:   ptr.String(""),
				},
			},
			isValid: false,
		},
		{
			name: "valid schedule",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(true),
					AutoUpdateStartTime: ptr.String("14:30"),
					AutoUpdateEndTime:   ptr.String("15:30"),
				},
			},
			isValid: true,
		},
		{
			name: "valid schedule (wrapped around midnight)",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(true),
					AutoUpdateStartTime: ptr.String("23:30"),
					AutoUpdateEndTime:   ptr.String("00:30"),
				},
			},
			isValid: true,
		},
		{
			name: "start time invalid",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(true),
					AutoUpdateStartTime: ptr.String("invalid"),
					AutoUpdateEndTime:   ptr.String("15:30"),
				},
			},
			isValid: false,
		},
		{
			name: "end time invalid",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(true),
					AutoUpdateStartTime: ptr.String("14:30"),
					AutoUpdateEndTime:   ptr.String("invalid"),
				},
			},
			isValid: false,
		},
		{
			name: "start time hour out of range",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(true),
					AutoUpdateStartTime: ptr.String("24:00"),
					AutoUpdateEndTime:   ptr.String("15:30"),
				},
			},
			isValid: false,
		},
		{
			name: "end time hour out of range",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(true),
					AutoUpdateStartTime: ptr.String("14:30"),
					AutoUpdateEndTime:   ptr.String("24:00"),
				},
			},
			isValid: false,
		},
		{
			name: "window is less than one hour",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(true),
					AutoUpdateStartTime: ptr.String("14:30"),
					AutoUpdateEndTime:   ptr.String("15:29"),
				},
			},
			isValid: false,
		},
		{
			name: "window is less than one hour (wrapped around midnight)",
			schedule: SoftwareAutoUpdateSchedule{
				SoftwareAutoUpdateConfig: SoftwareAutoUpdateConfig{
					AutoUpdateEnabled:   ptr.Bool(true),
					AutoUpdateStartTime: ptr.String("23:30"),
					AutoUpdateEndTime:   ptr.String("00:29"),
				},
			},
			isValid: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.schedule.WindowIsValid()
			if tc.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
