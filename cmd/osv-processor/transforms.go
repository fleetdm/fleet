package main

// transformVuln applies transformations and filters to OSV vulnerability data.
func transformVuln(packageName, cveID string, vuln *ProcessedVuln) (packages []string, modifiedVuln *ProcessedVuln) {
	// To completely ignore a CVE definition return nil
	// if cveID == "CVE-YYYY-XXXXX" {
	//     return nil, nil
	// }

	// Default: include the original package
	packages = []string{packageName}

	// Package expansion rules: Add related packages that should also get this CVE

	// Emacs CVEs (CVE-2024-39331, CVE-2024-53920, CVE-2025-1244, etc.)
	// Emacs vulnerabilities are in the Emacs Lisp runtime/interpreter shared across all packages.
	if packageName == "emacs" {
		packages = append(packages, "emacs-common", "emacs-el")
	}

	// CVE-specific modifications: modify vulnerability details for specific CVEs
	// if cveID == "CVE-YYYY-XXXXX" {
	//     modified := *vuln // Copy the vulnerability
	//     modified.Fixed = "corrected-version"
	//     return packages, &modified
	// }

	// If the vulnerability requires no modifications return original
	return packages, nil
}

// appendUbuntuBinaryPackages adds the binary package names supplied by
// Canonical's Ubuntu OSV records.
//
// Ubuntu OSV records are normally keyed by source package, while Fleet's
// software inventory contains installed binary package names. For example:
//
//	source package: libssh2
//	binary package: libssh2-1t64
//
// Keep the source package name for backwards compatibility and add all
// declared binary package names as additional artifact keys.
func appendUbuntuBinaryPackages(packages []string, affected Affected) []string {
	binaryCount := 0
	if affected.EcosystemSpecific != nil {
		binaryCount = len(affected.EcosystemSpecific.Binaries)
	}

	result := make([]string, 0, len(packages)+binaryCount)
	seen := make(map[string]struct{}, len(packages)+binaryCount)

	add := func(name string) {
		if name == "" {
			return
		}

		if _, exists := seen[name]; exists {
			return
		}

		seen[name] = struct{}{}
		result = append(result, name)
	}

	for _, packageName := range packages {
		add(packageName)
	}

	if affected.EcosystemSpecific != nil {
		for _, binary := range affected.EcosystemSpecific.Binaries {
			add(binary.BinaryName)
		}
	}

	return result
}
