// Package govulndb detects vulnerabilities in Go binaries using the Go vulnerability database
// (https://vuln.go.dev), the same data source govulncheck uses.
//
// Go binaries cannot be matched through NVD: a binary's name says nothing about its module,
// so the CPE path gave every binary named "air" Adobe AIR's CVEs. The Go database is keyed on
// module path (software.extension_id) and toolchain version (software.release) instead.
//
// Fleet does not reach vuln.go.dev directly. cmd/govulndb-mirror, run by a job in
// fleetdm/vulnerabilities, publishes the database as a release asset there, keeping a single
// egress point for restricted deployments. See README.md for the artifact contract.
//
// The exported constants and types below are the artifact contract, shared with the mirror so
// the two sides cannot drift apart.
package govulndb

import "time"

const (
	// softwareSource is the software source this package scans.
	softwareSource = "go_binaries"

	// SchemaVersion is the artifact schema this package reads. See README.md for the contract.
	SchemaVersion = "1"

	// FilePrefix is the prefix of a mirrored artifact's file name.
	FilePrefix = "govulndb-"

	// FileExt is the suffix of a mirrored artifact's file name.
	FileExt = ".json.gz"

	// StdlibModule is the Go vulnerability database's module path for standard-library
	// reports, matched against a binary's toolchain version rather than its own.
	StdlibModule = "stdlib"

	// develVersion is what the Go toolchain records for a working-tree build; there is no
	// release to compare against.
	develVersion = "(devel)"
)

// Artifact is the mirrored Go vulnerability database.
type Artifact struct {
	SchemaVersion string    `json:"schema_version"`
	Generated     time.Time `json:"generated"`
	// Modules maps a module path to its advisories, with the standard library under
	// StdlibModule. Toolchain advisories are not mirrored: they affect the build machine, not
	// the binary.
	Modules map[string][]Advisory `json:"modules"`
}

// Advisory is one Go vulnerability report's effect on one module.
type Advisory struct {
	// ID is the report ID, e.g. "GO-2024-2963".
	ID string `json:"id"`
	// CVEs are the report's CVE aliases. Reports without one are not mirrored: software_cve is
	// keyed on CVE ID.
	CVEs []string `json:"cves"`
	// Ranges are the affected version ranges with OSV events already paired. Versions carry
	// no leading "v".
	Ranges []VersionRange `json:"ranges"`
}

// VersionRange is a half-open affected range: Introduced <= v < Fixed. An empty Fixed means
// the report has no fixed version yet.
type VersionRange struct {
	Introduced string `json:"introduced,omitempty"`
	Fixed      string `json:"fixed,omitempty"`
}
