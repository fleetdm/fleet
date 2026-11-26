package externalrefs

import (
	"net/http"
	"net/http/httptest"
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdobeAcrobatReaderManifestVersion(t *testing.T) {
	const (
		caskVersion = "26.001.21662"
		caskURL     = "https://ardownload2.adobe.com/pub/adobe/reader/mac/AcrobatDC/2600121662/AcroRdrDC_2600121662_MUI.dmg"
		caskSHA     = "21821609bea4210e54b1844a1129cdafccea21f6d6d3d0797808f92e357d5705"
	)

	cases := []struct {
		name          string
		manifest      string
		manifestCode  int
		installerCode int
		wantVersion   string
		wantInstaller string
		wantSHA       string
		wantErr       bool
	}{
		{
			name:          "newer release with published installer",
			manifest:      "26.003.22110\n",
			installerCode: http.StatusOK,
			wantVersion:   "26.003.22110",
			wantInstaller: "/AcrobatDC/2600322110/AcroRdrDC_2600322110_MUI.dmg",
			wantSHA:       "no_check",
		},
		{
			name:        "same release keeps the cask",
			manifest:    caskVersion,
			wantVersion: caskVersion,
			wantSHA:     caskSHA,
		},
		{
			name:        "older release keeps the cask",
			manifest:    "25.001.21288",
			wantVersion: caskVersion,
			wantSHA:     caskSHA,
		},
		{
			name:          "installer not published yet keeps the cask",
			manifest:      "26.003.22110",
			installerCode: http.StatusNotFound,
			wantVersion:   caskVersion,
			wantSHA:       caskSHA,
		},
		{
			name:          "installer check fails",
			manifest:      "26.003.22110",
			installerCode: http.StatusInternalServerError,
			wantVersion:   caskVersion,
			wantSHA:       caskSHA,
			wantErr:       true,
		},
		{
			name:        "unexpected manifest contents",
			manifest:    "<html>error</html>",
			wantVersion: caskVersion,
			wantSHA:     caskSHA,
			wantErr:     true,
		},
		{
			name:         "manifest unavailable",
			manifestCode: http.StatusServiceUnavailable,
			wantVersion:  caskVersion,
			wantSHA:      caskSHA,
			wantErr:      true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/current_version.txt":
					if c.manifestCode != 0 {
						w.WriteHeader(c.manifestCode)
						return
					}
					_, _ = w.Write([]byte(c.manifest))
				default:
					require.Equal(t, http.MethodHead, r.Method)
					if c.installerCode == 0 {
						t.Errorf("unexpected installer request %s", r.URL.Path)
					}
					w.Header().Set("Content-Length", "738766348")
					w.WriteHeader(c.installerCode)
				}
			}))
			defer srv.Close()

			origManifest, origDMG := adobeReaderManifestURL, adobeReaderDMGURL
			t.Cleanup(func() { adobeReaderManifestURL, adobeReaderDMGURL = origManifest, origDMG })
			adobeReaderManifestURL = srv.URL + "/current_version.txt"
			adobeReaderDMGURL = srv.URL + "/AcrobatDC/%[1]s/AcroRdrDC_%[1]s_MUI.dmg"

			app := &maintained_apps.FMAManifestApp{Version: caskVersion, InstallerURL: caskURL, SHA256: caskSHA}
			got, err := AdobeAcrobatReaderManifestVersion(app)
			if c.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.NotNil(t, got)
			assert.Equal(t, c.wantVersion, got.Version)
			assert.Equal(t, c.wantSHA, got.SHA256)
			if c.wantInstaller != "" {
				assert.Equal(t, srv.URL+c.wantInstaller, got.InstallerURL)
			} else {
				assert.Equal(t, caskURL, got.InstallerURL)
			}
		})
	}
}

func TestAdobeVersionNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"26.003.22110", "26.001.21662", true},
		{"26.001.21662", "26.003.22110", false},
		{"26.001.21662", "26.001.21662", false},
		{"26.001.21700", "26.001.21662", true},
		{"27.0", "26.999.99999", true},
		{"26.001.021662", "26.001.21662", false},
		{"abc", "26.001.21662", false},
		{"26.003.22110", "", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, adobeVersionNewer(c.a, c.b), "adobeVersionNewer(%q, %q)", c.a, c.b)
	}
}
