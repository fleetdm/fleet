package constant

import "time"

const (
	// DefaultDirMode is the default file mode to apply to created directories.
	DefaultDirMode = 0o755
	// DefaultFileMode is the default file mode to apply to created files.
	DefaultFileMode = 0o600
	// DefaultWorldReadableFileMode is the default file mode to apply to files
	// that can be read by other processes.
	DefaultWorldReadableFileMode = 0o644
	// DefaultSystemdUnitMode is the required file mode to systemd unit files.
	DefaultSystemdUnitMode = DefaultWorldReadableFileMode
	// DesktopAppExecName is the name of Fleet's Desktop executable.
	//
	// We use fleet-desktop as name to properly identify the process when listing
	// running processes/tasks.
	DesktopAppExecName = "fleet-desktop"
	// OrbitNodeKeyFileName is the filename on disk where we write the orbit node key to
	OrbitNodeKeyFileName = "secret-orbit-node-key.txt"
	// OrbitEnrollMaxRetries is the max retries when doing an enroll request
	OrbitEnrollMaxRetries = 3
	// OrbitEnrollRetrySleep is the time duration to sleep between retries
	OrbitEnrollRetrySleep = 5 * time.Second
	// OsquerydName is the name of osqueryd binary
	// We use osqueryd as name to properly identify the process when listing
	// running processes/tasks.
	OsquerydName = "osqueryd"
	// OsqueryPidfile is the file containing the PID of the running osqueryd process
	OsqueryPidfile = "osquery.pid"
	// SystemServiceName is the name of Orbit system service
	// The service name is used by the OS service management framework
	SystemServiceName = "Fleet osquery"
)
