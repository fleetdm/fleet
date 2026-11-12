package update

import (
	"encoding/hex"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
)

// fleetdInstallerName is the product name of fleetd's MSI, which Installed apps shows until fleetd renames its entry.
const fleetdInstallerName = "Fleet osquery"

// isFleetdUninstallEntry returns whether a Windows Installed apps entry is fleetd's. Other apps are called "Fleet" too,
// so the name fleetd renames its entry to also needs Fleet's publisher. The installer's name is matched alone, as it
// always was.
func isFleetdUninstallEntry(displayName, publisher string) bool {
	switch displayName {
	case fleetdInstallerName:
		return true
	case constant.AgentDisplayName:
		return publisher == constant.AgentPublisher
	default:
		return false
	}
}

// packedProductCode returns a product code such as {AFA95216-3541-4281-BD30-8C16737ACBA1} in the form Windows Installer
// names its registry keys with: the first three groups reversed, then the two digits of each remaining byte swapped.
func packedProductCode(code string) (string, bool) {
	if len(code) != 38 || code[0] != '{' || code[37] != '}' {
		return "", false
	}
	groups := strings.Split(code[1:37], "-")
	if len(groups) != 5 || len(groups[0]) != 8 || len(groups[1]) != 4 || len(groups[2]) != 4 || len(groups[3]) != 4 {
		return "", false
	}
	if _, err := hex.DecodeString(strings.Join(groups, "")); err != nil {
		return "", false
	}
	var b strings.Builder
	for _, g := range groups[:3] {
		for i := len(g) - 1; i >= 0; i-- {
			b.WriteByte(g[i])
		}
	}
	rest := groups[3] + groups[4]
	for i := 0; i < len(rest); i += 2 {
		b.WriteByte(rest[i+1])
		b.WriteByte(rest[i])
	}
	return b.String(), true
}
