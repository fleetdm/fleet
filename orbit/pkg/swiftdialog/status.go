package swiftdialog

import (
	"os"

	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
)

// CanRun returns whether or not swiftDialog can run on the host.
func CanRun(rootDirPath string) bool {
	_, swiftDialogPath, _ := update.LocalTargetPaths(
		rootDirPath,
		"swiftDialog",
		update.SwiftDialogMacOSTarget,
	)

	if _, err := os.Stat(swiftDialogPath); err != nil {
		return false
	}

	return true
}
