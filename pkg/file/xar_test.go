package file

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

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
	tests := []struct {
		name             string
		file             string
		expectedName     string
		expectedVersion  string
		expectedBundleID string
	}{
		{
			name:             "1Password",
			file:             "distribution-1password.xml",
			expectedName:     "1Password.app",
			expectedVersion:  "8.10.34",
			expectedBundleID: "com.1password.1password",
		},
		{
			name:             "Chrome",
			file:             "distribution-chrome.xml",
			expectedName:     "Google Chrome.app",
			expectedVersion:  "126.0.6478.62",
			expectedBundleID: "com.google.Chrome",
		},
		{
			name:             "Microsoft Edge",
			file:             "distribution-edge.xml",
			expectedName:     "Microsoft Edge.app",
			expectedVersion:  "126.0.2592.56",
			expectedBundleID: "com.microsoft.edgemac",
		},
		{
			name:             "Firefox",
			file:             "distribution-firefox.xml",
			expectedName:     "Firefox.app",
			expectedVersion:  "99.0.0",
			expectedBundleID: "org.mozilla.firefox",
		},
		{
			name:             "fleetd",
			file:             "distribution-fleet.xml",
			expectedName:     "Fleet osquery",
			expectedVersion:  "42.0.0",
			expectedBundleID: "com.fleetdm.orbit",
		},
		{
			name:             "Go",
			file:             "distribution-go.xml",
			expectedName:     "Go",
			expectedVersion:  "go1.22.4",
			expectedBundleID: "org.golang.go",
		},
		{
			name:             "Microsoft Teams",
			file:             "distribution-microsoft-teams.xml",
			expectedName:     "Microsoft Teams.app",
			expectedVersion:  "24124.1412.2911.3341",
			expectedBundleID: "com.microsoft.teams2",
		},
		{
			name:             "Zoom",
			file:             "distribution-zoom.xml",
			expectedName:     "zoom.us.app",
			expectedVersion:  "6.0.11.35001",
			expectedBundleID: "us.zoom.xos",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawXML, err := os.ReadFile(filepath.Join("testdata", "distribution", tt.file))
			require.NoError(t, err)
			metadata, err := parseDistributionFile(rawXML)
			require.NoError(t, err)
			require.Equal(t, tt.expectedName, metadata.Name)
			require.Equal(t, tt.expectedVersion, metadata.Version)
			require.Equal(t, tt.expectedBundleID, metadata.BundleIdentifier)
		})
	}
}
