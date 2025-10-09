package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSoftware_populateBrowserField(t *testing.T) {
	tests := []struct {
		name            string
		software        Software
		expectedBrowser string
	}{
		{
			name: "chrome extension should populate browser field",
			software: Software{
				Source:       "chrome_extensions",
				ExtensionFor: "chrome",
			},
			expectedBrowser: "chrome",
		},
		{
			name: "firefox extension should populate browser field",
			software: Software{
				Source:       "firefox_addons",
				ExtensionFor: "firefox",
			},
			expectedBrowser: "firefox",
		},
		{
			name: "ie extension should populate browser field",
			software: Software{
				Source:       "ie_extensions",
				ExtensionFor: "ie",
			},
			expectedBrowser: "ie",
		},
		{
			name: "safari extension should populate browser field",
			software: Software{
				Source:       "safari_extensions",
				ExtensionFor: "safari",
			},
			expectedBrowser: "safari",
		},
		{
			name: "regular app should not populate browser field",
			software: Software{
				Source:       "apps",
				ExtensionFor: "",
			},
			expectedBrowser: "",
		},
		{
			name: "homebrew package should not populate browser field",
			software: Software{
				Source:       "homebrew_packages",
				ExtensionFor: "",
			},
			expectedBrowser: "",
		},
		{
			name: "vscode extension should not populate browser field",
			software: Software{
				Source:       "vscode_extensions",
				ExtensionFor: "",
			},
			expectedBrowser: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.software.populateBrowserField()
			require.Equal(t, tt.expectedBrowser, tt.software.Browser)
		})
	}
}

func TestSoftwareTitle_populateBrowserField(t *testing.T) {
	tests := []struct {
		name            string
		softwareTitle   SoftwareTitle
		expectedBrowser string
	}{
		{
			name: "chrome extension should populate browser field",
			softwareTitle: SoftwareTitle{
				Source:       "chrome_extensions",
				ExtensionFor: "chrome",
			},
			expectedBrowser: "chrome",
		},
		{
			name: "firefox extension should populate browser field",
			softwareTitle: SoftwareTitle{
				Source:       "firefox_addons",
				ExtensionFor: "firefox",
			},
			expectedBrowser: "firefox",
		},
		{
			name: "ie extension should populate browser field",
			softwareTitle: SoftwareTitle{
				Source:       "ie_extensions",
				ExtensionFor: "ie",
			},
			expectedBrowser: "ie",
		},
		{
			name: "safari extension should populate browser field",
			softwareTitle: SoftwareTitle{
				Source:       "safari_extensions",
				ExtensionFor: "safari",
			},
			expectedBrowser: "safari",
		},
		{
			name: "regular app should not populate browser field",
			softwareTitle: SoftwareTitle{
				Source:       "apps",
				ExtensionFor: "",
			},
			expectedBrowser: "",
		},
		{
			name: "homebrew package should not populate browser field",
			softwareTitle: SoftwareTitle{
				Source:       "homebrew_packages",
				ExtensionFor: "",
			},
			expectedBrowser: "",
		},
		{
			name: "vscode extension should not populate browser field",
			softwareTitle: SoftwareTitle{
				Source:       "vscode_extensions",
				ExtensionFor: "",
			},
			expectedBrowser: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.softwareTitle.populateBrowserField()
			require.Equal(t, tt.expectedBrowser, tt.softwareTitle.Browser)
		})
	}
}

func TestSoftwareTitleListResult_populateBrowserField(t *testing.T) {
	tests := []struct {
		name                    string
		softwareTitleListResult SoftwareTitleListResult
		expectedBrowser         string
	}{
		{
			name: "chrome extension should populate browser field",
			softwareTitleListResult: SoftwareTitleListResult{
				Source:       "chrome_extensions",
				ExtensionFor: "chrome",
			},
			expectedBrowser: "chrome",
		},
		{
			name: "firefox extension should populate browser field",
			softwareTitleListResult: SoftwareTitleListResult{
				Source:       "firefox_addons",
				ExtensionFor: "firefox",
			},
			expectedBrowser: "firefox",
		},
		{
			name: "ie extension should populate browser field",
			softwareTitleListResult: SoftwareTitleListResult{
				Source:       "ie_extensions",
				ExtensionFor: "ie",
			},
			expectedBrowser: "ie",
		},
		{
			name: "safari extension should populate browser field",
			softwareTitleListResult: SoftwareTitleListResult{
				Source:       "safari_extensions",
				ExtensionFor: "safari",
			},
			expectedBrowser: "safari",
		},
		{
			name: "regular app should not populate browser field",
			softwareTitleListResult: SoftwareTitleListResult{
				Source:       "apps",
				ExtensionFor: "",
			},
			expectedBrowser: "",
		},
		{
			name: "homebrew package should not populate browser field",
			softwareTitleListResult: SoftwareTitleListResult{
				Source:       "homebrew_packages",
				ExtensionFor: "",
			},
			expectedBrowser: "",
		},
		{
			name: "vscode extension should not populate browser field",
			softwareTitleListResult: SoftwareTitleListResult{
				Source:       "vscode_extensions",
				ExtensionFor: "",
			},
			expectedBrowser: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.softwareTitleListResult.populateBrowserField()
			require.Equal(t, tt.expectedBrowser, tt.softwareTitleListResult.Browser)
		})
	}
}
