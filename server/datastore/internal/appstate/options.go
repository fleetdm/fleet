package appstate

import "github.com/kolide/fleet/server/kolide"

// Options is the set of builtin osquery options that should be populated in
// the datastore
func Options() []struct {
	Name     string
	Value    interface{}
	Type     kolide.OptionType
	ReadOnly bool
} {

	return []struct {
		Name     string
		Value    interface{}
		Type     kolide.OptionType
		ReadOnly bool
	}{
		// These options are read only, attempting to modify one of these will
		// raise an error
		{"disable_distributed", false, kolide.OptionTypeBool, kolide.ReadOnly},
		{"distributed_plugin", "tls", kolide.OptionTypeString, kolide.ReadOnly},
		{"distributed_tls_read_endpoint", "/api/v1/osquery/distributed/read", kolide.OptionTypeString, kolide.ReadOnly},
		{"distributed_tls_write_endpoint", "/api/v1/osquery/distributed/write", kolide.OptionTypeString, kolide.ReadOnly},
		{"pack_delimiter", "/", kolide.OptionTypeString, kolide.ReadOnly},
		// These options may be modified by an admin user
		{"aws_access_key_id", nil, kolide.OptionTypeString, kolide.NotReadOnly},
		{"aws_firehose_period", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"aws_firehose_stream", nil, kolide.OptionTypeString, kolide.NotReadOnly},
		{"aws_kinesis_period", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"aws_kinesis_random_partition_key", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"aws_kinesis_stream", nil, kolide.OptionTypeString, kolide.NotReadOnly},
		{"aws_profile_name", nil, kolide.OptionTypeString, kolide.NotReadOnly},
		{"aws_region", nil, kolide.OptionTypeString, kolide.NotReadOnly},
		{"aws_secret_access_key", nil, kolide.OptionTypeString, kolide.NotReadOnly},
		{"aws_sts_arn_role", nil, kolide.OptionTypeString, kolide.NotReadOnly},
		{"aws_sts_region", nil, kolide.OptionTypeString, kolide.NotReadOnly},
		{"aws_sts_session_name", nil, kolide.OptionTypeString, kolide.NotReadOnly},
		{"aws_sts_timeout", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"buffered_log_max", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"decorations_top_level", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"disable_caching", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"disable_database", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"disable_decorators", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"disable_events", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"disable_kernel", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"disable_logging", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"disable_tables", nil, kolide.OptionTypeString, kolide.NotReadOnly},
		{"distributed_interval", 10, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"distributed_tls_max_attempts", 3, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"enable_foreign", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"enable_monitor", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"ephemeral", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"events_expiry", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"events_max", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"events_optimize", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"host_identifier", nil, kolide.OptionTypeString, kolide.NotReadOnly},
		{"logger_event_type", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"logger_mode", nil, kolide.OptionTypeString, kolide.NotReadOnly},
		{"logger_path", nil, kolide.OptionTypeString, kolide.NotReadOnly},
		{"logger_plugin", "tls", kolide.OptionTypeString, kolide.NotReadOnly},
		{"logger_secondary_status_only", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"logger_syslog_facility", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"logger_tls_compress", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"logger_tls_endpoint", "/api/v1/osquery/log", kolide.OptionTypeString, kolide.NotReadOnly},
		{"logger_tls_max", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"logger_tls_period", 10, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"pack_refresh_interval", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"read_max", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"read_user_max", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"schedule_default_interval", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"schedule_splay_percent", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"schedule_timeout", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"utc", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"value_max", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
		{"verbose", nil, kolide.OptionTypeBool, kolide.NotReadOnly},
		{"worker_threads", nil, kolide.OptionTypeInt, kolide.NotReadOnly},
	}
}
