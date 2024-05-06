package file

import (
	_ "embed"
	"path/filepath"
)

//go:embed scripts/install_pkg.sh
var installPkgScript string

//go:embed scripts/install_msi.ps1
var installMsiScript string

//go:embed scripts/install_exe.ps1
var installExeScript string

//go:embed scripts/install_deb.sh
var installDebScript string

// GetInstallScript returns a script that can be used to install the given file
func GetInstallScript(filename string) string {
	switch ext := filepath.Ext(filename); ext {
	case ".msi":
		return installMsiScript
	case ".deb":
		return installDebScript
	case ".pkg":
		return installPkgScript
	case ".exe":
		return installExeScript
	default:
		return ""
	}
}

//go:embed scripts/remove_exe.ps1
var removeExeScript string

//go:embed scripts/remove_pkg.sh
var removePkgScript string

//go:embed scripts/remove_msi.ps1
var removeMsiScript string

//go:embed scripts/remove_deb.sh
var removeDebScript string

// GetRemoveScript returns a script that can be used to remove the given file
func GetRemoveScript(filename string) string {
	switch ext := filepath.Ext(filename); ext {
	case ".msi":
		return removeMsiScript
	case ".deb":
		return removeDebScript
	case ".pkg":
		return removePkgScript
	case ".exe":
		return removeExeScript
	default:
		return ""
	}
}
