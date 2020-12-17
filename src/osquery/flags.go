package osquery

func fleetFlags(hostname string) []string {
	return []string{
		"--tls_hostname=" + hostname,
		"--enroll_tls_endpoint=/api/v1/osquery/enroll",
		"--config_plugin=tls",
		"--config_tls_endpoint=/api/v1/osquery/config",
		"--disable_distributed=false",
		"--distributed_plugin=tls",
		"--distributed_tls_max_attempts=10",
		"--distributed_tls_read_endpoint=/api/v1/osquery/distributed/read",
		"--distributed_tls_write_endpoint=/api/v1/osquery/distributed/write",
		"--logger_plugin=tls",
		"--logger_tls_endpoint=/api/v1/osquery/log",
		"--disable_carver=false",
		"--carver_start_endpoint=/api/v1/osquery/carve/begin",
		"--carver_continue_endpoint=/api/v1/osquery/carve/block",
		"--carver_block_size=2000000",
	}
}
