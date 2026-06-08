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
	MaxSoftwareBatchSize     int64 = 25 * units.MiB // Takes multiple installers, with scripts and queries
	MaxMDMCommandSize        int64 = 2 * units.MiB
	// MaxMultiScriptQuerySize, sets a max size for payloads that take multiple scripts and SQL queries.
	MaxMultiScriptQuerySize int64 = 5 * units.MiB
	MaxMicrosoftMDMSize     int64 = 2 * units.MiB

	// DefaultMaxOsqueryLogWriteSize is the default request body size limit
	// applied to /api/osquery/log when osquery.allow_body_auth_fallback is
	// true (legacy body-auth mode). Operators can override via the
	// osquery.max_log_write_body_size config. In header-auth mode
	// (allow_body_auth_fallback=false) this limit does not apply; the
	// route inherits the global request body size limit.
	DefaultMaxOsqueryLogWriteSize int64 = 10 * units.MiB
	// DefaultMaxOsqueryDistributedWriteSize is the same as
	// DefaultMaxOsqueryLogWriteSize but for /api/osquery/distributed/write.
	DefaultMaxOsqueryDistributedWriteSize int64 = 5 * units.MiB
)
