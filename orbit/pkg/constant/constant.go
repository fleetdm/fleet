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
	// OrbitEnrollMaxRetries is the max number of retries when doing an enroll request.
	// We set it to 6 to allow the retry backoff to take effect.
	OrbitEnrollMaxRetries = 6
	// OrbitEnrollBackoffMultiplier is the multiplier to use for backing off between enroll retries.
	OrbitEnrollBackoffMultiplier = 2
	// OrbitEnrollRetrySleep is the duration to sleep between enroll retries.
	OrbitEnrollRetrySleep = 10 * time.Second
	// OsquerydName is the name of osqueryd binary
	// We use osqueryd as name to properly identify the process when listing
	// running processes/tasks.
	OsquerydName = "osqueryd"
	// OsqueryPidfile is the file containing the PID of the running osqueryd process
	OsqueryPidfile = "osquery.pid"
	// OsqueryEnrollSecretFileName is the filename on disk where we write
	// the orbit enroll secret.
	OsqueryEnrollSecretFileName = "secret.txt"
	// SystemServiceName is the name of Orbit system service
	// The service name is used by the OS service management framework
	SystemServiceName = "Fleet osquery"
	// FleetTLSClientCertificateFileName is the name of the TLS client certificate file
	// used when connecting to the Fleet server.
	FleetTLSClientCertificateFileName = "fleet_client.crt"
	// FleetTLSClientKeyFileName is the name of the TLS client private key file
	// used when connecting to the Fleet server.
	FleetTLSClientKeyFileName = "fleet_client.key"
	// UpdateTLSClientCertificateFileName is the name of the TLS client certificate file
	// used when connecting to the update server.
	UpdateTLSClientCertificateFileName = "update_client.crt"
	// UpdateTLSClientKeyFileName is the name of the TLS client private key file
	// used when connecting to the update server.
	UpdateTLSClientKeyFileName = "update_client.key"
	// SilenceEnrollLogErrorEnvVer is an environment variable name for disabling enroll log errors
	SilenceEnrollLogErrorEnvVar = "FLEETD_SILENCE_ENROLL_ERROR"
	// ServerOverridesFileName is the name of the file in the root directory
	// that specifies the override configuration fetched from the server.
	ServerOverridesFileName = "server-overrides.json"
)
