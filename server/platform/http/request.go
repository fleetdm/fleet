package http

import "github.com/docker/go-units"

// WE need the default to be a var, since we want it configurable, and to avoid misses in the future we set it to the config value on startup/serve.
var MaxRequestBodySize int64 = units.MiB // Default which is 1 MiB

const (
	// MaxMultipartFormSize represents how big the in memory elements is when parsing a multipart form data set,
	// anything above that limit (primarily files) will be written to temp disk files
	MaxMultipartFormSize int64 = 1 * units.MiB
)
