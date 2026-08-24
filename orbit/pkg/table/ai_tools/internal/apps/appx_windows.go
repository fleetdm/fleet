//go:build windows

package apps

import (
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

// MSIX/Appx packages are why the uninstall-key scan is not enough on its own.
// They never write an entry under CurrentVersion\Uninstall — that is the
// convention of legacy Win32 installers — so a Store-distributed app is
// invisible to that scan no matter which hive it walks. AI apps ship this way
// (ChatGPT desktop among them).
//
// Detection reads directories rather than the registry. The
// AppModel\Repository\Packages key that used to list packages is a Windows 8 /
// early-Windows-10 artifact and is absent on current Windows, where package
// state lives in the StateRepository database under
// C:\ProgramData\Microsoft\Windows\AppRepository — a locked SQLite file owned by
// a service, and not something a collector should open. The directories below
// need no such machinery: they are named by package full name and package family
// name, which carry the identity outright.

// appxUserDataSubdir is a package's per-user state directory, relative to a home
// directory and named by package family name.
var appxUserDataSubdir = filepath.Join("AppData", "Local", "Packages")

// scanAppx adds the AI apps installed as MSIX/Appx packages to c.
//
// It shares the caller's collector rather than building its own, so an app also
// found in the uninstall keys keeps that entry: a real DisplayVersion and
// InstallLocation beat a package directory.
func scanAppx(c *appCollector, homesList []homes.Home) {
	userDirs := make([]string, 0, len(homesList))
	for _, h := range homesList {
		userDirs = append(userDirs, filepath.Join(h.Dir, appxUserDataSubdir))
	}
	scanAppxDirs(c, appxInstallRoot(), userDirs)
}

// appxInstallRoot returns the packaged-app install root, normally
// "C:\Program Files\WindowsApps".
func appxInstallRoot() string {
	return appxInstallRootFrom(
		os.Getenv("ProgramW6432"),
		os.Getenv("ProgramFiles"),
		os.Getenv("SystemDrive"),
	)
}
