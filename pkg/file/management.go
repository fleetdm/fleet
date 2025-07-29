package file

import (
	_ "embed"
	"regexp"
)

//go:embed scripts/install_pkg.sh
var installPkgScript string

//go:embed scripts/install_msi.ps1
var installMsiScript string

//go:embed scripts/install_deb.sh
var installDebScript string

//go:embed scripts/install_rpm.sh
var installRPMScript string

// GetInstallScript returns a script that can be used to install the given extension
func GetInstallScript(extension string) string {
	switch extension {
	case "msi":
		return installMsiScript
	case "deb":
		return installDebScript
	case "rpm":
		return installRPMScript
	case "pkg":
		return installPkgScript
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

//go:embed scripts/remove_rpm.sh
var removeRPMScript string

// GetRemoveScript returns a script that can be used to remove an
// installer with the given extension.
func GetRemoveScript(extension string) string {
	switch extension {
	case "msi":
		return removeMsiScript
	case "deb":
		return removeDebScript
	case "rpm":
		return removeRPMScript
	case "pkg":
		return removePkgScript
	case "exe":
		return removeExeScript
	default:
		return ""
	}
}

//go:embed scripts/uninstall_pkg.sh
var uninstallPkgScript string

//go:embed scripts/uninstall_msi.ps1
var uninstallMsiScript string

//go:embed scripts/uninstall_msi_with_upgrade_code.ps1
var UninstallMsiWithUpgradeCodeScript string

var PackageIDRegex = regexp.MustCompile(`((("\$PACKAGE_ID")|(\$PACKAGE_ID))(?P<suffix>\W|$))|(("\${PACKAGE_ID}")|(\${PACKAGE_ID}))`)
var UpgradeCodeRegex = regexp.MustCompile(`((("\$UPGRADE_CODE")|(\$UPGRADE_CODE))(?P<suffix>\W|$))|(("\${UPGRADE_CODE}")|(\${UPGRADE_CODE}))`)

//go:embed scripts/uninstall_deb.sh
var uninstallDebScript string

//go:embed scripts/uninstall_rpm.sh
var uninstallRPMScript string

// GetUninstallScript returns a script that can be used to uninstall a
// software item with the given extension.
func GetUninstallScript(extension string) string {
	switch extension {
	case "msi":
		return uninstallMsiScript
	case "deb":
		return uninstallDebScript
	case "rpm":
		return uninstallRPMScript
	case "pkg":
		return uninstallPkgScript
	default:
		return ""
	}
}
