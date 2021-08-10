package osquery

import (
	"net/url"
	"path"
)

// FleetFlags is the set of flags to pass to osquery when connecting to Fleet.
func FleetFlags(fleetURL *url.URL) []string {
	hostname, prefix := fleetURL.Host, fleetURL.Path
	return []string{
		"--tls_hostname=" + hostname,
		"--enroll_tls_endpoint=" + path.Join(prefix, "/api/v1/osquery/enroll"),
		"--config_plugin=tls",
		"--config_tls_endpoint=" + path.Join(prefix, "/api/v1/osquery/config"),
		// Osquery defaults config_refresh to 0 which is probably not ideal for
		// a client connected to Fleet. Users can always override this in the
		// config they serve via Fleet.
		"--config_refresh=60",
		"--disable_distributed=false",
		"--distributed_plugin=tls",
		"--distributed_tls_max_attempts=10",
		"--distributed_tls_read_endpoint=" + path.Join(prefix, "/api/v1/osquery/distributed/read"),
		"--distributed_tls_write_endpoint=" + path.Join(prefix, "/api/v1/osquery/distributed/write"),
		"--logger_plugin=tls",
		"--logger_tls_endpoint=" + path.Join(prefix, "/api/v1/osquery/log"),
		"--disable_carver=false",
		"--carver_start_endpoint=" + path.Join(prefix, "/api/v1/osquery/carve/begin"),
		"--carver_continue_endpoint=" + path.Join(prefix, "/api/v1/osquery/carve/block"),
		"--carver_block_size=2000000",
	}
}
