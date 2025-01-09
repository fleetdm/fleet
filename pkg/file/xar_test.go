package file

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckPKGSignature(t *testing.T) {
	read := func(name string) []byte {
		b, err := os.ReadFile(name)
		require.NoError(t, err)
		return b
	}
	testCases := []struct {
		in  []byte
		out error
	}{
		{in: []byte{}, out: io.EOF},
		{
			in:  read("./testdata/invalid.tar.gz"),
			out: ErrInvalidType,
		},
		{
			in:  read("./testdata/unsigned.pkg"),
			out: ErrNotSigned,
		},
		{
			in:  read("./testdata/signed.pkg"),
			out: nil,
		},
		{
			out: errors.New("decompressing TOC: unexpected EOF"),
			in:  read("./testdata/wrong-toc.pkg"),
		},
	}

	for _, c := range testCases {
		r := bytes.NewReader(c.in)
		err := CheckPKGSignature(r)
		if c.out != nil {
			require.ErrorContains(t, err, c.out.Error())
		} else {
			require.NoError(t, err)
		}
	}
}

func TestParseRealDistributionFiles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		file               string
		expectedName       string
		expectedVersion    string
		expectedBundleID   string
		expectedPackageIDs []string
	}{
		{
			file:               "distribution-1password.xml",
			expectedName:       "1Password.app",
			expectedVersion:    "8.10.34",
			expectedBundleID:   "com.1password.1password",
			expectedPackageIDs: []string{"com.1password.1password"},
		},
		{
			file:               "distribution-chrome.xml",
			expectedName:       "Google Chrome.app",
			expectedVersion:    "126.0.6478.62",
			expectedBundleID:   "com.google.Chrome",
			expectedPackageIDs: []string{"com.google.Chrome"},
		},
		{
			file:               "distribution-edge.xml",
			expectedName:       "Microsoft Edge.app",
			expectedVersion:    "126.0.2592.56",
			expectedBundleID:   "com.microsoft.edgemac",
			expectedPackageIDs: []string{"com.microsoft.edgemac"},
		},
		{
			file:               "distribution-firefox.xml",
			expectedName:       "Firefox.app",
			expectedVersion:    "99.0",
			expectedBundleID:   "org.mozilla.firefox",
			expectedPackageIDs: []string{"org.mozilla.firefox"},
		},
		{
			file:               "distribution-fleet.xml",
			expectedName:       "Fleet osquery",
			expectedVersion:    "42.0.0",
			expectedBundleID:   "com.fleetdm.orbit",
			expectedPackageIDs: []string{"com.fleetdm.orbit.base.pkg"},
		},
		{
			file:               "distribution-go.xml",
			expectedName:       "Go",
			expectedVersion:    "go1.22.4",
			expectedBundleID:   "org.golang.go",
			expectedPackageIDs: []string{"org.golang.go"},
		},
		{
			file:             "distribution-microsoft-teams.xml",
			expectedName:     "Microsoft Teams.app",
			expectedVersion:  "24124.1412.2911.3341",
			expectedBundleID: "com.microsoft.teams2",
			expectedPackageIDs: []string{
				"com.microsoft.teams2", "com.microsoft.package.Microsoft_AutoUpdate.app",
				"com.microsoft.MSTeamsAudioDevice",
			},
		},
		{
			file:               "distribution-zoom.xml",
			expectedName:       "zoom.us.app",
			expectedVersion:    "6.0.11.35001",
			expectedBundleID:   "us.zoom.xos",
			expectedPackageIDs: []string{"us.zoom.pkg.videomeeting"},
		},
		{
			file:             "distribution-acrobatreader.xml",
			expectedName:     "Adobe Acrobat Reader.app",
			expectedVersion:  "24.002.20857",
			expectedBundleID: "com.adobe.Reader",
			expectedPackageIDs: []string{
				"com.adobe.acrobat.DC.reader.app.pkg.MUI", "com.adobe.acrobat.DC.reader.appsupport.pkg.MUI",
				"com.adobe.acrobat.reader.DC.reader.app.pkg.MUI", "com.adobe.armdc.app.pkg",
			},
		},
		{
			file:               "distribution-airtame.xml",
			expectedName:       "Airtame.app",
			expectedVersion:    "4.10.1",
			expectedBundleID:   "com.airtame.airtame-application",
			expectedPackageIDs: []string{"com.airtame.airtame-application"},
		},
		{
			file:             "distribution-boxdrive.xml",
			expectedName:     "Box.app",
			expectedVersion:  "2.38.173",
			expectedBundleID: "com.box.desktop",
			expectedPackageIDs: []string{
				"com.box.desktop.installer.desktop", "com.box.desktop.installer.local.appsupport",
				"com.box.desktop.installer.autoupdater", "com.box.desktop.installer.osxfuse",
			},
		},
		{
			file:             "distribution-iriunwebcam.xml",
			expectedName:     "IriunWebcam.app",
			expectedVersion:  "2.8.8",
			expectedBundleID: "com.iriun.macwebcam",
			// Note: "com.iriun.pkg.multicam" is part of the installer package, but it is not actually installed by default.
			// We can't reliably determine which packages are installed by the installer, so we just list all of them.
			expectedPackageIDs: []string{"com.iriun.pkg.webcam.tmp", "com.iriun.pkg.multicam"},
		},
		{
			file:             "distribution-microsoftexcel.xml",
			expectedName:     "Microsoft Excel.app",
			expectedVersion:  "16.86",
			expectedBundleID: "com.microsoft.Excel",
			expectedPackageIDs: []string{
				"com.microsoft.package.Microsoft_Excel.app", "com.microsoft.package.Microsoft_AutoUpdate.app",
				"com.microsoft.pkg.licensing",
			},
		},
		{
			file:             "distribution-microsoftword.xml",
			expectedName:     "Microsoft Word.app",
			expectedVersion:  "16.86",
			expectedBundleID: "com.microsoft.Word",
			expectedPackageIDs: []string{
				"com.microsoft.package.Microsoft_Word.app", "com.microsoft.package.Microsoft_AutoUpdate.app",
				"com.microsoft.pkg.licensing",
			},
		},
		{
			file:             "distribution-miscrosoftpowerpoint.xml",
			expectedName:     "Microsoft PowerPoint.app",
			expectedVersion:  "16.86",
			expectedBundleID: "com.microsoft.Powerpoint",
			expectedPackageIDs: []string{
				"com.microsoft.package.Microsoft_PowerPoint.app", "com.microsoft.package.Microsoft_AutoUpdate.app",
				"com.microsoft.pkg.licensing",
			},
		},
		{
			file:               "distribution-ringcentral.xml",
			expectedName:       "RingCentral.app",
			expectedVersion:    "24.1.32.9774",
			expectedBundleID:   "com.ringcentral.glip",
			expectedPackageIDs: []string{"com.ringcentral.glip"},
		},
		{
			file:               "distribution-zoom-full.xml",
			expectedName:       "Zoom Workplace",
			expectedVersion:    "6.1.1.36333",
			expectedBundleID:   "us.zoom.xos",
			expectedPackageIDs: []string{"us.zoom.pkg.videomeeting"},
		},
		{
			file:               "test-zero-installkbytes.xml",
			expectedName:       "ZeroInstallSize.app",
			expectedVersion:    "1.2.3",
			expectedBundleID:   "com.bozo.zeroinstallsize",
			expectedPackageIDs: []string{"com.bozo.zeroinstallsize.app"},
		},
		{
			file:               "distribution-sentinelone.xml",
			expectedName:       "SentinelOne",
			expectedVersion:    "24.3.2.7753",
			expectedBundleID:   "com.sentinelone.pkg.sentinel-agent",
			expectedPackageIDs: []string{"com.sentinelone.pkg.sentinel-agent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			rawXML, err := os.ReadFile(filepath.Join("testdata", "distribution", tt.file))
			require.NoError(t, err)
			metadata, err := parseDistributionFile(rawXML)
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.expectedPackageIDs, metadata.PackageIDs)
			require.Equal(t, tt.expectedName, metadata.Name)
			require.Equal(t, tt.expectedVersion, metadata.Version)
			require.Equal(t, tt.expectedBundleID, metadata.BundleIdentifier)
		})
	}
}

func TestParsePackageInfoFiles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		file               string
		expectedName       string
		expectedVersion    string
		expectedBundleID   string
		expectedPackageIDs []string
	}{
		{
			file:               "packageInfo-oktaVerify.xml",
			expectedName:       "Okta Verify.app",
			expectedVersion:    "9.27.0",
			expectedBundleID:   "com.okta.mobile",
			expectedPackageIDs: []string{"com.okta.mobile"},
		},
		{
			file:             "packageInfo-iriunWebcam.xml",
			expectedName:     "IriunWebcam.app",
			expectedVersion:  "2.8.10",
			expectedBundleID: "com.iriun.macwebcam",
			expectedPackageIDs: []string{
				"com.iriun.macwebcam", "com.iriun.macwebcam.extension4", "com.iriun.macwebcam.extension",
				"com.iriun.mic",
			},
		},
		{
			file:               "packageInfo-scriptOnly.xml",
			expectedName:       "HelloWorld",
			expectedVersion:    "1.2.3",
			expectedBundleID:   "com.mygreatcompany.pkg.HelloWorld",
			expectedPackageIDs: []string{"com.mygreatcompany.pkg.HelloWorld"},
		},
		{
			file:               "packageInfo-empty.xml",
			expectedName:       "",
			expectedVersion:    "",
			expectedBundleID:   "",
			expectedPackageIDs: []string{},
		},
		{
			file:               "packageInfo-versionOnly.xml",
			expectedName:       "",
			expectedVersion:    "test-version",
			expectedBundleID:   "",
			expectedPackageIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			rawXML, err := os.ReadFile(filepath.Join("testdata", "packageInfo", tt.file))
			require.NoError(t, err)
			metadata, err := parsePackageInfoFile(rawXML)
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.expectedPackageIDs, metadata.PackageIDs)
			assert.Equal(t, tt.expectedName, metadata.Name)
			assert.Equal(t, tt.expectedVersion, metadata.Version)
			assert.Equal(t, tt.expectedBundleID, metadata.BundleIdentifier)
		})
	}
}

func TestIsValidAppFilePath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"baz.app", true},
		{"foo/bar/baz.app", false},
		{"Applications/baz.app", true},
		{"Applications/foo/baz.app", false},
		{"foo/baz.app", false},
		{"baz.txt", true},
		{"Applications/baz.txt", false},
	}

	for _, test := range tests {
		_, ok := isValidAppFilePath(test.input)
		require.Equal(t, test.expected, ok, test.input)
	}
}
