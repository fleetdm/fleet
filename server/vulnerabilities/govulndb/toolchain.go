package govulndb

import (
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// toolchainSources are the software sources that report Go toolchain installs. deb and rpm
// packages are left to OVAL: distributions backport fixes without changing the upstream
// version, so matching their version against the Go database would report fixed CVEs.
var toolchainSources = []string{"homebrew_packages", "pacman_packages", "programs"}

// toolchainNameMatch narrows the toolchain iterator in SQL. MySQL's REGEXP ignores case and
// knows nothing about sources, so IsToolchainInstall still makes the exact call.
const toolchainNameMatch = `^(go(@.*)?|Go Programming Language .*)$`

// IsToolchainInstall reports whether software is an install of the Go toolchain itself, which
// is matched against the Go vulnerability database rather than NVD.
func IsToolchainInstall(software *fleet.Software) bool {
	switch software.Source {
	case "homebrew_packages":
		return software.Name == "go" || strings.HasPrefix(software.Name, "go@")
	case "pacman_packages":
		return software.Name == "go"
	case "programs":
		return strings.HasPrefix(software.Name, "Go Programming Language ")
	}
	return false
}

// toolchainInstallVersion canonicalizes a toolchain install's version into semver.
func toolchainInstallVersion(software *fleet.Software) (string, bool) {
	version := software.Version
	switch software.Source {
	case "homebrew_packages":
		// Homebrew appends "_N" when it rebuilds a formula without an upstream version change.
		version, _, _ = strings.Cut(version, "_")
	case "pacman_packages":
		// Fleet now stores only the upstream version, but rows ingested by older versions still
		// carry Arch's [epoch:]upstream-pkgrel form.
		if _, after, ok := strings.Cut(version, ":"); ok {
			version = after
		}
		if i := strings.LastIndex(version, "-"); i >= 0 {
			version = version[:i]
		}
	case "programs":
		// The installer names itself after the Go tag ("... amd64 go1.22rc1"). Its MSI version
		// must be numeric, so it cannot represent a release candidate.
		if fields := strings.Fields(software.Name); len(fields) > 0 && strings.HasPrefix(fields[len(fields)-1], "go") {
			version = fields[len(fields)-1]
		}
	}
	return toolchainVersion(version)
}

// matchToolchainInstall returns the stdlib and toolchain advisories affecting a Go toolchain
// install. Unlike a binary, upgrading the toolchain is the fix, so the fixed version is kept.
func matchToolchainInstall(software *fleet.Software, artifact *Artifact) []fleet.SoftwareVulnerability {
	if !IsToolchainInstall(software) {
		return nil
	}
	version, ok := toolchainInstallVersion(software)
	if !ok {
		return nil
	}

	vulns := newVulnSet(software.ID)
	vulns.addAffected(artifact.Modules[StdlibModule], version, upgradeTo(""))
	vulns.addAffected(artifact.Modules[ToolchainModule], version, upgradeTo(""))
	return vulns.vulns
}
