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
