package file

import (
	_ "embed"
	"os"
)

type InstallerType string

const (
	InstallerTypeMsi InstallerType = "msi"
	InstallerTypeDeb InstallerType = "deb"
	InstallerTypePkg InstallerType = "pkg"
	InstallerTypeExe InstallerType = "exe"
)

//go:embed scripts/install_pkg.sh
var installPkgScript string

//go:embed scripts/install_msi.ps1
var installMsiScript string

//go:embed scripts/install_exe.ps1
var installExeScript string

//go:embed scripts/install_deb.sh
var installDebScript string

// GetInstallScript returns a script that can be used to install the given
// installer based on the provided type
func GetInstallScript(installerType InstallerType, installerPath string) string {
	var rawScript string

	switch installerType {
	case InstallerTypeMsi:
		rawScript = installMsiScript
	case InstallerTypeDeb:
		rawScript = installDebScript
	case InstallerTypePkg:
		rawScript = installPkgScript
	case InstallerTypeExe:
		rawScript = installExeScript
	default:
		return ""
	}

	return os.Expand(rawScript, scriptMapper(installerPath))
}

//go:embed scripts/remove_exe.ps1
var removeExeScript string

//go:embed scripts/remove_pkg.sh
var removePkgScript string

//go:embed scripts/remove_msi.ps1
var removeMsiScript string

//go:embed scripts/remove_deb.sh
var removeDebScript string

// GetRemoveScript returns a script that can be used to remove the given
// installer based on the provided type
func GetRemoveScript(installerType InstallerType, installerPath string) string {
	var rawScript string

	switch installerType {
	case InstallerTypeMsi:
		rawScript = removeMsiScript
	case InstallerTypeDeb:
		rawScript = removeDebScript
	case InstallerTypePkg:
		rawScript = removePkgScript
	case InstallerTypeExe:
		rawScript = removeExeScript
	default:
		return ""
	}

	return os.Expand(rawScript, scriptMapper(installerPath))
}

func scriptMapper(installerPath string) func(string) string {
	return func(placeholder string) string {
		if placeholder == "INSTALLER_PATH" {
			return installerPath
		}
		return ""
	}
}
