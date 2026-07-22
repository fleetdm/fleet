package sigverify

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstallerFilename(t *testing.T) {
	cases := []struct {
		name   string
		remote string
		want   string
	}{
		{"msi", "GoogleChromeStandaloneEnterprise64.msi", "installer.msi"},
		{"exe", "Setup.exe", "installer.exe"},
		{"uppercase extension normalized", "Firefox.PKG", "installer.pkg"},
		{"mixed case extension", "app.Dmg", "installer.dmg"},
		{"zip", "slack.zip", "installer.zip"},
		{"msix", "Teams.msix", "installer.msix"},
		{"unknown extension dropped", "payload.bin", "installer"},
		{"no extension", "download", "installer"},
		{"empty filename", "", "installer"},
		{"path traversal attempt", "../../../etc/passwd", "installer"},
		{"traversal with known extension keeps only the extension", "../../evil.msi", "installer.msi"},
		{"dot file", ".msi", "installer.msi"},
		{"double extension keeps only the last", "app.tar.gz", "installer"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, InstallerFilename(tc.remote))
		})
	}
}
