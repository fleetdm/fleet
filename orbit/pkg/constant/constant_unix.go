//go:build !windows
// +build !windows

package constant

const (
	// DefaultExecutableMode is the default file mode to apply to created
	// executable files.
	DefaultExecutableMode = 0o755
)
