package constant

const (
	// DefaultDirMode is the default file mode to apply to created directories.
	DefaultDirMode = 0o755
	// DefaultFileMode is the default file mode to apply to created files.
	DefaultFileMode = 0o600
	// DesktopAppExecName is the name of Fleet's Desktop executable.
	//
	// We use fleet-desktop as name to properly identify the process when listing
	// running processes/tasks.
	DesktopAppExecName = "fleet-desktop"
)
