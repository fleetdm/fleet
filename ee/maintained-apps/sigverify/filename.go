package sigverify

import (
	"path/filepath"
	"strings"
)

// InstallerFilename returns the fixed local filename a downloaded installer
// must be saved under before signature verification. Verification dispatches
// on the file extension only, and the remote filename
// (Content-Disposition/URL) is attacker-influenced and must never reach path
// construction — so the result is "installer" plus the remote extension
// matched against known installer formats, always a literal. Unknown
// extensions save as extensionless "installer", which verification treats as
// an unrecognized format rather than silently skipping.
func InstallerFilename(remoteFilename string) string {
	ext := filepath.Ext(remoteFilename)
	for _, known := range []string{".exe", ".msi", ".msix", ".appx", ".dll", ".cab", ".zip", ".pkg", ".mpkg", ".dmg"} {
		if strings.EqualFold(ext, known) {
			return "installer" + known
		}
	}
	return "installer"
}
