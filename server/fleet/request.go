package fleet

import "github.com/docker/go-units"

// This file declares max request body size limits for individual or grouped endpoints, it's together for easy review and re-usage.
// The only outlier is the MaxSoftwareInstallerSize which is in the installersize context package, as it's used in multiple places outside of request handling, and to avoid circular imports.

const (
	MaxSpecSize              int64 = 25 * units.MiB
	MaxFleetdErrorReportSize int64 = 5 * units.MiB
	MaxScriptSize            int64 = 1.5 * units.MiB // 1.5 to allow for roughly 1MB content, and B64 encoding
	MaxBatchScriptSize       int64 = 25 * units.MiB
	MaxProfileSize           int64 = 1.5 * units.MiB // 1.5 to allow for roughly 1MB content, and B64 encoding
	MaxBatchProfileSize      int64 = 25 * units.MiB
	MaxEULASize              int64 = 25 * units.MiB
	MaxMDMCommandSize        int64 = 2 * units.MiB
	// MaxMultiScriptQuerySize, sets a max size for payloads that take multiple scripts and SQL queries.
	MaxMultiScriptQuerySize int64 = 5 * units.MiB
	MaxMicrosoftMDMSize     int64 = 2 * units.MiB
)
