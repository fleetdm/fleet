package constant

const (
	// DefaultDirMode is the default file mode to apply to created directories.
	DefaultDirMode = 0o755
	// DefaultFileMode is the default file mode to apply to created files.
	DefaultFileMode = 0o600
	// DefaultSystemdUnitMode is the required file mode to systemd unit files.
	DefaultSystemdUnitMode = 0o644
	// DesktopAppExecName is the name of Fleet's Desktop executable.
	//
	// We use fleet-desktop as name to properly identify the process when listing
	// running processes/tasks.
	DesktopAppExecName = "fleet-desktop"

	// OsquerydName is the name of osqueryd binary
	//
	// We use osqueryd as name to properly identify the process when listing
	// running processes/tasks.
	OsquerydName = "osqueryd"

	// OsqueryPidfile is the file containing the PID of the running osqueryd process
	OsqueryPidfile = "osquery.pid"

	// SystemServiceName is the name of Orbit system service
	// The service name is used by the OS service management framework
	SystemServiceName = "Fleet osquery"
)
