// Package govulndb detects vulnerabilities in Go binaries using the Go vulnerability database
// (https://vuln.go.dev), the same data source govulncheck uses.
//
// Go binaries cannot be matched through NVD. A binary's name says nothing about the module it
// was built from, so the generic CPE path gave every binary named "air" the Adobe AIR CPE and
// all of its CVEs. The Go database is keyed on module path instead, which fleetd reports and
// Fleet stores in software.extension_id, and on the Go toolchain version, stored in
// software.release.
//
// Fleet does not reach vuln.go.dev directly. A job in fleetdm/vulnerabilities pulls the
// database and publishes it as a release asset that this package downloads, which keeps a
// single egress point for restricted deployments. See README.md for the artifact contract.
package govulndb

import "time"

const (
	// softwareSource is the software source this package scans.
	softwareSource = "go_binaries"

	// schemaVersion is the artifact schema this package reads. See README.md for the contract.
	schemaVersion = "1"

	// filePrefix is the prefix of a mirrored artifact's file name.
	filePrefix = "govulndb-"

	// fileExt is the suffix of a mirrored artifact's file name.
	fileExt = ".json.gz"

	// stdlibModule is the module path the Go vulnerability database uses for its
	// standard-library reports. Advisories under it are matched against a binary's Go
	// toolchain version rather than its own version.
	stdlibModule = "stdlib"

	// develVersion is the version the Go toolchain records for a binary built from a
	// working tree rather than a tagged module release. There is nothing to compare it
	// against, so those rows are skipped.
	develVersion = "(devel)"
)

// Artifact is the mirrored Go vulnerability database.
type Artifact struct {
	SchemaVersion string    `json:"schema_version"`
	Generated     time.Time `json:"generated"`
	// Modules maps a Go module path to the advisories affecting it, with the standard
	// library under stdlibModule. Toolchain advisories are not mirrored: they affect the
	// `go` command on the build machine, not the binary it produced.
	Modules map[string][]Advisory `json:"modules"`
}

// Advisory is one Go vulnerability report's effect on one module.
type Advisory struct {
	// ID is the report ID, e.g. "GO-2024-2963".
	ID string `json:"id"`
	// CVEs are the report's CVE aliases. A report without one is not mirrored: Fleet's
	// software_cve rows and every CVE metadata join are keyed on a CVE ID.
	CVEs []string `json:"cves"`
	// Ranges are the affected version ranges, with the publisher's OSV events already
	// paired up. Versions carry no leading "v".
	Ranges []VersionRange `json:"ranges"`
}

// VersionRange is a half-open affected range: Introduced <= v < Fixed. An empty Fixed means
// the report has no fixed version yet.
type VersionRange struct {
	Introduced string `json:"introduced,omitempty"`
	Fixed      string `json:"fixed,omitempty"`
}
