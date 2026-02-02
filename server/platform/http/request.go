package http

import "github.com/docker/go-units"

// WE need the default to be a var, since we want it configurable, and to avoid misses in the future we set it to the config value on startup/serve.
var MaxRequestBodySize int64 = units.MiB // Default which is 1 MiB

const (
	MaxFleetdErrorReportSize int64 = 5 * units.MiB
	// MaxMultipartFormSize represents how big the in memory elements is when parsing a multipart form data set,
	// anything above that limit (primarily files) will be written to temp disk files
	MaxMultipartFormSize     int64 = 1 * units.MiB
	MaxScriptSize            int64 = 1 * units.MiB
	MaxBatchScriptSize       int64 = 25 * units.MiB
	MaxProfileSize           int64 = 1 * units.MiB
	MaxBatchProfileSize      int64 = 25 * units.MiB
	MaxEULASize              int64 = 500 * units.MiB
	MaxMDMCommandSize        int64 = 2 * units.MiB
	MaxSoftwareInstallerSize int64 = 10 * units.GiB
	// MaxMultiScriptQuerySize, sets a max size for payloads that take multiple scripts and SQL queries.
	MaxMultiScriptQuerySize int64 = 5 * units.MiB
	MaxMicrosoftMDMSize     int64 = 2 * units.MiB
	MaxSpecSize             int64 = 25 * units.MiB
)
