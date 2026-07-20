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
	// MaxProfileSizeErrMsg reports the ~1MB content limit that MaxProfileSize
	// enforces (the extra 0.5 MiB is base64 headroom, not usable content).
	MaxProfileSizeErrMsg = "maximum configuration profile file size is 1 MB"
	MaxMDMAssetSize          int64 = 1.5 * units.MiB // 1.5 to allow for roughly 1MB content, and B64 encoding
	MaxEULASize              int64 = 25 * units.MiB
	MaxSoftwareBatchSize     int64 = 25 * units.MiB // Takes multiple installers, with scripts and queries
	MaxMDMCommandSize        int64 = 2 * units.MiB
	// MaxMultiScriptQuerySize, sets a max size for payloads that take multiple scripts and SQL queries.
	MaxMultiScriptQuerySize int64 = 5 * units.MiB
	MaxMicrosoftMDMSize     int64 = 2 * units.MiB
	// MaxSSOCallbackSize bounds the body of the unauthenticated SSO callback
	// endpoints (regular and MDM). The body carries a base64-encoded
	// SAMLResponse; legitimate responses are well under 50 KiB even after
	// base64 inflation, so 256 KiB leaves generous headroom for large
	// enterprise IdP responses while keeping pre-auth attacks surface small.
	MaxSSOCallbackSize int64 = 256 * units.KiB
	// MaxAppleMDMRequestBodySize bounds Apple MDM check-in and command-result
	// request bodies. Results are stored in a MEDIUMTEXT column (max 16,777,215
	// bytes), so the limit must not exceed that boundary.
	MaxAppleMDMRequestBodySize int64 = (16 * units.MiB) - 1

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
