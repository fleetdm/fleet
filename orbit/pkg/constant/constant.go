package constant

const (
	// DefaultDirMode is the default file mode to apply to created directories.
	DefaultDirMode = 0o755
	// DefaultFileMode is the default file mode to apply to created files.
	DefaultFileMode = 0o600
)

// ExecutableExtension returns the extension used for executables on the
// provided platform.
func ExecutableExtension(platform string) string {
	switch platform {
	case "windows":
		return ".exe"
	default:
		return ""
	}
}
