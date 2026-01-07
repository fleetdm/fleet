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
