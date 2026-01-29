// This module exists solely for automated vulnerability scanning of inlined dependencies.
// It is NOT compiled into Fleet. Do not add any Go code to this directory.
//
// These dependencies have been copied/inlined into Fleet's codebase. This go.mod file
// allows tools like GitHub Dependabot and govulncheck to detect vulnerabilities in
// the specific versions that were copied.
//
// IMPORTANT: When updating inlined dependencies, update the versions here to match.

module github.com/fleetdm/fleet/v4/third_party/vuln-check

go 1.25.6

require (
	// NanoMDM - Apple MDM server (server/mdm/nanomdm/)
	// Copied: January 2024, Updated: November 2024
	// Using latest tagged version that includes our copied commit
	github.com/micromdm/nanomdm v0.9.0

	// nanoDEP - Apple DEP API tools (server/mdm/nanodep/)
	// Copied: February 2024
	github.com/micromdm/nanodep v0.4.0

	// SCEP - Certificate enrollment (server/mdm/scep/)
	// Copied: February 2024, Updated: September 2024
	github.com/micromdm/scep/v2 v2.3.0

	// goose - Database migrations (server/goose/)
	// Copied: December 2023
	github.com/pressly/goose/v3 v3.17.0

	// NVD Tools - Vulnerability database tools (server/vulnerabilities/nvd/tools/)
	// Copied: April 2024
	github.com/facebookincubator/nvdtools v0.1.5

	// go-paniclog - Panic output capture (orbit/pkg/go-paniclog/)
	// Copied: January 2024 (repo last updated Aug 2019)
	github.com/virtuald/go-paniclog v0.0.0-20190812204905-43a7fa316459

	// mockimpl - Mock stub generator (server/mock/mockimpl/)
	// Copied: December 2023
	// Based on github.com/groob/mockimpl which forked github.com/josharian/impl
	github.com/josharian/impl v1.4.0

	// gon - macOS notarization (orbit/pkg/packaging/macos_notarize.go)
	// Only statusHuman struct copied from v0.2.3
	github.com/mitchellh/gon v0.2.3

	// relic - XAR file parsing (pkg/file/xar.go)
	// Copied: April 2023
	github.com/sassoftware/relic v7.2.1+incompatible

	// poc-mdm-server - Windows MDM demo (tools/mdm/windows/poc-mdm-server/)
	// Forked from oscartbeaumont/windows_mdm
	github.com/oscartbeaumont/windows_mdm v0.0.0-20210615145659-e52e28e50db7
)
