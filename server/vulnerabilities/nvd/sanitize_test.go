package nvd

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestVariations(t *testing.T) {
	variationsTestCases := []struct {
		software          fleet.Software
		vendorVariations  []string
		productVariations []string
	}{
		{
			software:          fleet.Software{Name: "1Password – Password Manager", Version: "2.3.8", Source: "chrome_extensions"},
			productVariations: []string{"1password"},
		},

		{
			software:          fleet.Software{Name: "AdBlock — best ad blocker", Version: "5.1.1", Source: "chrome_extensions"},
			productVariations: []string{"adblock"},
		},
		{
			software:          fleet.Software{Name: "Adblock Plus - free ad blocker", Version: "3.14.2", Source: "chrome_extensions"},
			productVariations: []string{"adblockplus", "adblock_plus"},
		},
		{
			software:          fleet.Software{Name: "uBlock Origin", Version: "1.44.4", Source: "chrome_extensions"},
			productVariations: []string{"ublockorigin", "ublock_origin"},
		},
		{
			software:          fleet.Software{Name: "Adobe Acrobat DC (64-bit)", Version: "22.002.20212", Source: "programs", Vendor: "Adobe"},
			vendorVariations:  []string{"adobe"},
			productVariations: []string{"acrobatdc", "acrobat_dc"},
		},
		{
			software:          fleet.Software{Name: "Bing", Version: "1.3", Source: "firefox_addons"},
			productVariations: []string{"bing"},
		},
		{
			software:          fleet.Software{Name: "Brave", Version: "105.1.43.93", Source: "programs", Vendor: "Brave Software Inc"},
			vendorVariations:  []string{"brave", "brave_software_inc", "bravesoftwareinc"},
			productVariations: []string{"brave"},
		},
		{
			software:          fleet.Software{Name: "Docker Desktop", Version: "4.12.0", Source: "programs", Vendor: "Docker Inc."},
			vendorVariations:  []string{"docker", "docker_inc.", "dockerinc."},
			productVariations: []string{"desktop", "docker_desktop", "dockerdesktop"},
		},
		{
			software:          fleet.Software{Name: "Dropbox", Version: "157.4.4808", Source: "programs", Vendor: "Dropbox, Inc."},
			vendorVariations:  []string{"dropbox,_inc.", "dropbox,inc.", "dropbox,"},
			productVariations: []string{"dropbox"},
		},
		{
			software:          fleet.Software{Name: "DuckDuckGo", Version: "1.1", Source: "firefox_addons"},
			productVariations: []string{"duckduckgo"},
		},
		{
			software:          fleet.Software{Name: "Git", Version: "2.37.1", Source: "programs", Vendor: "The Git Development Community"},
			vendorVariations:  []string{"git", "thegitdevelopmentcommunity", "development", "community", "the_git_development_community"},
			productVariations: []string{"git"},
		},
		{
			software:          fleet.Software{Name: "Google Chrome", Version: "105.0.5195.127", Source: "programs", Vendor: "Google LLC"},
			vendorVariations:  []string{"google_llc", "googlellc", "google", "llc"},
			productVariations: []string{"chrome", "google_chrome", "googlechrome"},
		},
		{
			software:          fleet.Software{Name: "Microsoft Edge", Version: "105.0.1343.50", Source: "programs", Vendor: "Microsoft Corporation"},
			vendorVariations:  []string{"microsoft", "microsoft_corporation", "microsoftcorporation"},
			productVariations: []string{"edge", "microsoft_edge", "microsoftedge"},
		},
		{
			software:          fleet.Software{Name: "Microsoft OneDrive", Version: "22.181.0828.0002", Source: "programs", Vendor: "Microsoft Corporation"},
			vendorVariations:  []string{"microsoft_corporation", "microsoftcorporation", "microsoft"},
			productVariations: []string{"onedrive", "microsoft_onedrive", "microsoftonedrive"},
		},
		{
			software:          fleet.Software{Name: "Microsoft Visual Studio Code (User)", Version: "1.71.2", Source: "programs", Vendor: "Microsoft Corporation"},
			vendorVariations:  []string{"microsoft", "microsoft_corporation", "microsoftcorporation"},
			productVariations: []string{"visualstudiocode", "visual_studio_code", "microsoft_visual_studio_code", "microsoftvisualstudiocode"},
		},
		{
			software:          fleet.Software{Name: "Mozilla Firefox (x64 en-US)", Version: "105.0.1", Source: "programs", Vendor: "Mozilla"},
			vendorVariations:  []string{"mozilla"},
			productVariations: []string{"firefox"},
		},
		{
			software:          fleet.Software{Name: "Oracle VM VirtualBox 6.1.38", Version: "6.1.38", Source: "programs", Vendor: "Oracle Corporation"},
			vendorVariations:  []string{"oracle", "oracle_corporation", "oraclecorporation"},
			productVariations: []string{"vmvirtualbox", "vm_virtualbox", "oracle_vm_virtualbox", "oraclevmvirtualbox"},
		},
		{
			software:          fleet.Software{Name: "Python 3.10.6 (64-bit)", Version: "3.10.6150.0", Source: "programs", Vendor: "Python Software Foundation"},
			vendorVariations:  []string{"python", "python_software_foundation", "pythonsoftwarefoundation"},
			productVariations: []string{"python"},
		},
		{
			software:          fleet.Software{Name: "VLC media player", Version: "3.0.17.4", Source: "programs", Vendor: "VideoLAN"},
			vendorVariations:  []string{"videolan"},
			productVariations: []string{"vlcmediaplayer", "vlc_media_player"},
		},
		{
			software:          fleet.Software{Name: "Visual Studio Community 2022", Version: "17.2.5", Source: "programs", Vendor: "Microsoft Corporation"},
			vendorVariations:  []string{"microsoft", "microsoft_corporation", "microsoftcorporation"},
			productVariations: []string{"visualstudiocommunity", "visual_studio_community"},
		},
		{
			software:          fleet.Software{Name: "uBlock Origin", Version: "1.44.0", Source: "chrome_extensions"},
			productVariations: []string{"ublockorigin", "ublock_origin"},
		},
		{
			software:          fleet.Software{Name: "Adobe Acrobat Reader DC.app", Version: "22.002.20191", BundleIdentifier: "com.adobe.Reader", Source: "apps"},
			vendorVariations:  []string{"adobe", "reader"},
			productVariations: []string{"acrobatreaderdc", "acrobat_reader_dc"},
		},
		{
			software:          fleet.Software{Name: "Adobe Lightroom.app", Version: "5.5", BundleIdentifier: "com.adobe.mas.lightroomCC", Source: "apps"},
			vendorVariations:  []string{"adobe", "mas", "lightroomcc"},
			productVariations: []string{"lightroom"},
		},
		{
			software:          fleet.Software{Name: "Finder.app", Version: "12.5", BundleIdentifier: "com.apple.finder", Source: "apps"},
			vendorVariations:  []string{"apple", "finder"},
			productVariations: []string{"finder"},
		},
		{
			software:          fleet.Software{Name: "Firefox.app", Version: "105.0.1", BundleIdentifier: "org.mozilla.firefox", Source: "apps"},
			vendorVariations:  []string{"mozilla", "firefox"},
			productVariations: []string{"firefox"},
		},
		{
			software:          fleet.Software{Name: "Google Chrome.app", Version: "105.0.5195.125", BundleIdentifier: "com.google.Chrome", Source: "apps"},
			vendorVariations:  []string{"chrome", "google"},
			productVariations: []string{"chrome"},
		},
		{
			software:          fleet.Software{Name: "Microsoft Excel.app", Version: "16.65", BundleIdentifier: "com.microsoft.Excel", Source: "apps"},
			vendorVariations:  []string{"microsoft", "excel"},
			productVariations: []string{"excel"},
		},
		{
			software:          fleet.Software{Name: "OneDrive.app", Version: "22.186.0904", BundleIdentifier: "com.microsoft.OneDrive-mac", Source: "apps"},
			vendorVariations:  []string{"microsoft", "onedrive-mac"},
			productVariations: []string{"onedrive"},
		},
		{
			software:          fleet.Software{Name: "Python.app", Version: "3.10.7", BundleIdentifier: "org.python.python", Source: "apps"},
			vendorVariations:  []string{"python"},
			productVariations: []string{"python"},
		},
		{
			software:          fleet.Software{Name: "Python.app", Version: "3.8.9", BundleIdentifier: "com.apple.python3", Source: "apps"},
			vendorVariations:  []string{"apple", "python3"},
			productVariations: []string{"python"},
		},
		{
			software:          fleet.Software{Name: "ms-python.python", Version: "3.8.9", BundleIdentifier: "", Source: "vscode_extensions", Vendor: "Microsoft"},
			vendorVariations:  []string{"microsoft", "ms-python"},
			productVariations: []string{"python", "ms-python.python"},
		},
	}

	for _, tc := range variationsTestCases {
		tc := tc
		require.ElementsMatch(t, tc.productVariations, productVariations(&tc.software), tc.software)
		require.ElementsMatch(t, tc.vendorVariations, vendorVariations(&tc.software), tc.software)
	}
}

func TestSanitizedSoftwareName(t *testing.T) {
	t.Run("removes arch from name", func(t *testing.T) {
		testCases := []struct {
			software fleet.Software
			expected string
		}{
			{
				software: fleet.Software{
					Name:    "Adobe Acrobat DC (64-bit)",
					Version: "22.002.20212",
					Vendor:  "Adobe",
					Source:  "programs",
				},
				expected: "acrobat dc",
			},
			{
				software: fleet.Software{
					Name:    "Mozilla Firefox (x64)",
					Version: "105.0.1",
					Vendor:  "Mozilla",
					Source:  "programs",
				},
				expected: "firefox",
			},
			{
				software: fleet.Software{
					Name:    "Python (64-bit)",
					Version: "3.10.6150.0",
					Vendor:  "Python Software Foundation",
					Source:  "programs",
				},
				expected: "python",
			},
		}

		for _, tc := range testCases {
			tc := tc
			actual := sanitizeSoftwareName(&tc.software)
			require.Equal(t, tc.expected, actual)
		}
	})

	t.Run("removes version from the name", func(t *testing.T) {
		testCases := []struct {
			software fleet.Software
			expected string
		}{
			{
				software: fleet.Software{
					Name:    "Oracle VM VirtualBox 6.1.38",
					Version: "6.1.38",
					Vendor:  "Oracle Corporation",
					Source:  "programs",
				},
				expected: "oracle vm virtualbox",
			},
			{
				software: fleet.Software{
					Name:    "Python 3.10.6 (64-bit)",
					Version: "3.10.6150.0",
					Vendor:  "Python Software Foundation",
					Source:  "programs",
				},
				expected: "python",
			},
		}

		for _, tc := range testCases {
			tc := tc
			actual := sanitizeSoftwareName(&tc.software)
			require.Equal(t, tc.expected, actual)
		}
	})

	t.Run("removes any extra comments", func(t *testing.T) {
		testCases := []struct {
			software fleet.Software
			expected string
		}{
			{
				software: fleet.Software{
					Name:    "1Password – Password Manager",
					Version: "2.3.8",
					Source:  "chrome_extensions",
				},
				expected: "1password",
			},
			{
				software: fleet.Software{
					Name:    "Adblock Plus - free ad blocker",
					Version: "3.14.2",
					Source:  "chrome_extensions",
				},
				expected: "adblock plus",
			},
			{
				software: fleet.Software{
					Name:    "AdBlock — best ad blocker",
					Version: "5.1.1",
					Vendor:  "",
					Source:  "chrome_extensions",
				},
				expected: "adblock",
			},
		}

		for _, tc := range testCases {
			tc := tc
			actual := sanitizeSoftwareName(&tc.software)
			require.Equal(t, tc.expected, actual)
		}
	})

	t.Run("removes any language codes", func(t *testing.T) {
		testCases := []struct {
			software fleet.Software
			expected string
		}{
			{
				software: fleet.Software{
					Name:    "Mozilla Firefox (x64 en-US)",
					Version: "105.0.1",
					Vendor:  "Mozilla",
					Source:  "programs",
				},
				expected: "firefox",
			},
		}

		for _, tc := range testCases {
			tc := tc
			actual := sanitizeSoftwareName(&tc.software)
			require.Equal(t, tc.expected, actual)
		}
	})

	t.Run("removes any () and its contents", func(t *testing.T) {
		testCases := []struct {
			software fleet.Software
			expected string
		}{
			{
				software: fleet.Software{
					Name:    "Microsoft Visual Studio Code (User)",
					Version: "1.71.2",
					Vendor:  "Microsoft Corporation",
					Source:  "programs",
				},
				expected: "microsoft visual studio code",
			},
		}

		for _, tc := range testCases {
			tc := tc
			actual := sanitizeSoftwareName(&tc.software)
			require.Equal(t, tc.expected, actual)
		}
	})

	t.Run("removes .app and bundle parts from the name", func(t *testing.T) {
		testCases := []struct {
			software fleet.Software
			expected string
		}{
			{
				software: fleet.Software{
					Name:             "Google Chrome.app",
					Version:          "105.0.5195.125",
					BundleIdentifier: "com.google.Chrome",
					Source:           "apps",
				},
				expected: "chrome",
			},
			{
				software: fleet.Software{
					Name:             "Microsoft Excel.app",
					Version:          "16.65",
					BundleIdentifier: "com.microsoft.Excel",
					Source:           "apps",
				},
				expected: "excel",
			},
			{
				software: fleet.Software{
					Name:             "TextEdit.app",
					Version:          "1.17",
					BundleIdentifier: "com.apple.TextEdit",
					Source:           "apps",
				},
				expected: "textedit",
			},
			{
				software: fleet.Software{
					Name:             "Firefox.app",
					Version:          "105.0.1",
					BundleIdentifier: "org.mozilla.firefox",
					Source:           "apps",
				},
				expected: "firefox",
			},
		}
		for _, tc := range testCases {
			tc := tc
			actual := sanitizeSoftwareName(&tc.software)
			require.Equal(t, tc.expected, actual)
		}
	})
}
