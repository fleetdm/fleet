package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type AgentOptions struct {
	// Config is the base config options.
	Config json.RawMessage `json:"config"`
	// Overrides includes any platform-based overrides.
	Overrides AgentOptionsOverrides `json:"overrides,omitempty"`
	// CommandLineStartUpFlags are the osquery CLI_FLAGS
	CommandLineStartUpFlags json.RawMessage `json:"command_line_flags,omitempty"`
	// Extensions are the orbit managed extensions
	Extensions json.RawMessage `json:"extensions,omitempty"`
}

type AgentOptionsOverrides struct {
	// Platforms is a map from platform name to the config override.
	Platforms map[string]json.RawMessage `json:"platforms,omitempty"`
}

func (o *AgentOptions) ForPlatform(platform string) json.RawMessage {
	// Return matching platform override if available.
	if opt, ok := o.Overrides.Platforms[platform]; ok {
		return opt
	}

	// Otherwise return base config for team.
	return o.Config
}

// ValidateJSONAgentOptions validates the given raw JSON bytes as an Agent
// Options payload. It ensures that all fields are known and have valid values.
// The validation always uses the most recent Osquery version that is available
// at the time of the Fleet release.
func ValidateJSONAgentOptions(ctx context.Context, ds Datastore, rawJSON json.RawMessage, isPremium bool) error {
	var opts AgentOptions
	if err := JSONStrictDecode(bytes.NewReader(rawJSON), &opts); err != nil {
		return err
	}

	if len(opts.CommandLineStartUpFlags) > 0 {
		var flags osqueryCommandLineFlags
		if err := JSONStrictDecode(bytes.NewReader(opts.CommandLineStartUpFlags), &flags); err != nil {
			return fmt.Errorf("command-line flags: %w", err)
		}
	}

	if len(opts.Config) > 0 {
		if err := validateJSONAgentOptionsSet(opts.Config); err != nil {
			return fmt.Errorf("common config: %w", err)
		}
	}

	for platform, platformOpts := range opts.Overrides.Platforms {
		if len(platformOpts) > 0 {
			if err := validateJSONAgentOptionsSet(platformOpts); err != nil {
				return fmt.Errorf("%s platform config: %w", platform, err)
			}
		}
	}

	if len(opts.Extensions) > 0 {
		if err := validateJSONAgentOptionsExtensions(ctx, ds, opts.Extensions, isPremium); err != nil {
			return err
		}
	}

	return nil
}

func validateJSONAgentOptionsExtensions(ctx context.Context, ds Datastore, optsExtensions json.RawMessage, isPremium bool) error {
	var extensions map[string]ExtensionInfo
	if err := json.Unmarshal(optsExtensions, &extensions); err != nil {
		return fmt.Errorf("unmarshal extensions: %w", err)
	}
	for _, extensionInfo := range extensions {
		if !isPremium && len(extensionInfo.Labels) != 0 {
			// Setting labels settings in the extensions config is premium only.
			return ErrMissingLicense
		}
		for _, labelName := range extensionInfo.Labels {
			switch _, err := ds.GetLabelSpec(ctx, labelName); {
			case err == nil:
				// OK
			case IsNotFound(err):
				// Label does not exist, fail the request.
				return fmt.Errorf("Label %q does not exist", labelName)
			default:
				return fmt.Errorf("get label by name: %w", err)
			}
		}
	}
	return nil
}

// JSON definition of the available configuration options in osquery.
// See https://osquery.readthedocs.io/en/stable/deployment/configuration/#configuration-specification
//
// NOTE: Update the following line with the version used for validation.
// Current version: 5.5.1
type osqueryAgentOptions struct {
	Options osqueryOptions `json:"options"`

	// Schedule is allowed as top-level key but we don't validate its value.
	// See https://github.com/fleetdm/fleet/issues/7871#issuecomment-1265531018
	Schedule json.RawMessage `json:"schedule"`

	// Packs is allowed as top-level key but we don't validate its value.
	// See https://github.com/fleetdm/fleet/issues/7871#issuecomment-1265531018
	Packs json.RawMessage `json:"packs"`

	FilePaths    map[string][]string `json:"file_paths"`
	FileAccesses []string            `json:"file_accesses"`
	// Documentation for the following 2 fields is "hidden" in osquery's FIM page:
	// https://osquery.readthedocs.io/en/stable/deployment/file-integrity-monitoring/
	FilePathsQuery map[string][]string `json:"file_paths_query"`
	ExcludePaths   map[string][]string `json:"exclude_paths"`

	YARA struct {
		Signatures map[string][]string `json:"signatures"`
		FilePaths  map[string][]string `json:"file_paths"`
		// Documentation for signature_urls is "hidden" in osquery's YARA page:
		// https://osquery.readthedocs.io/en/stable/deployment/yara/#retrieving-yara-rules-at-runtime
		SignatureURLs []string `json:"signature_urls"`
	} `json:"yara"`

	PrometheusTargets struct {
		Timeout int      `json:"timeout"`
		URLs    []string `json:"urls"`
	} `json:"prometheus_targets"`

	Views map[string]string `json:"views"`

	Decorators struct {
		Load     []string            `json:"load"`
		Always   []string            `json:"always"`
		Interval map[string][]string `json:"interval"`
	} `json:"decorators"`

	AutoTableConstruction map[string]struct {
		Query    string   `json:"query"`
		Path     string   `json:"path"`
		Columns  []string `json:"columns"`
		Platform string   `json:"platform"`
	} `json:"auto_table_construction"`

	Events struct {
		DisableSubscribers []string `json:"disable_subscribers"`
		// NOTE: documentation seems to imply that there is also an EnableSubscribers
		// field, but it is not explicitly shown in the example (nor is the name
		// explicitly given). Found out in the code that this is the case:
		// https://github.com/osquery/osquery/blob/bf697df445c5612522407781f86addb4b3d13221/osquery/events/eventfactory.cpp
		EnableSubscribers []string `json:"enable_subscribers"`
	} `json:"events"`
}

// NOTE: generate automatically with `go run ./tools/osquery-agent-options/main.go`
type osqueryOptions struct {
	AuditAllowAcceptSocketEvents        bool   `json:"audit_allow_accept_socket_events"`
	AuditAllowApparmorEvents            bool   `json:"audit_allow_apparmor_events"`
	AuditAllowConfig                    bool   `json:"audit_allow_config"`
	AuditAllowFailedSocketEvents        bool   `json:"audit_allow_failed_socket_events"`
	AuditAllowFimEvents                 bool   `json:"audit_allow_fim_events"`
	AuditAllowForkProcessEvents         bool   `json:"audit_allow_fork_process_events"`
	AuditAllowKillProcessEvents         bool   `json:"audit_allow_kill_process_events"`
	AuditAllowNullAcceptSocketEvents    bool   `json:"audit_allow_null_accept_socket_events"`
	AuditAllowProcessEvents             bool   `json:"audit_allow_process_events"`
	AuditAllowSeccompEvents             bool   `json:"audit_allow_seccomp_events"`
	AuditAllowSelinuxEvents             bool   `json:"audit_allow_selinux_events"`
	AuditAllowSockets                   bool   `json:"audit_allow_sockets"`
	AuditAllowUserEvents                bool   `json:"audit_allow_user_events"`
	AuditBacklogLimit                   int32  `json:"audit_backlog_limit"`
	AuditBacklogWaitTime                int32  `json:"audit_backlog_wait_time"`
	AuditForceReconfigure               bool   `json:"audit_force_reconfigure"`
	AuditForceUnconfigure               bool   `json:"audit_force_unconfigure"`
	AuditPersist                        bool   `json:"audit_persist"`
	AugeasLenses                        string `json:"augeas_lenses"`
	AwsAccessKeyId                      string `json:"aws_access_key_id"`
	AwsDebug                            bool   `json:"aws_debug"`
	AwsEnableProxy                      bool   `json:"aws_enable_proxy"`
	AwsFirehoseEndpoint                 string `json:"aws_firehose_endpoint"`
	AwsFirehosePeriod                   uint64 `json:"aws_firehose_period"`
	AwsFirehoseStream                   string `json:"aws_firehose_stream"`
	AwsKinesisDisableLogStatus          bool   `json:"aws_kinesis_disable_log_status"`
	AwsKinesisEndpoint                  string `json:"aws_kinesis_endpoint"`
	AwsKinesisPeriod                    uint64 `json:"aws_kinesis_period"`
	AwsKinesisRandomPartitionKey        bool   `json:"aws_kinesis_random_partition_key"`
	AwsKinesisStream                    string `json:"aws_kinesis_stream"`
	AwsProfileName                      string `json:"aws_profile_name"`
	AwsProxyHost                        string `json:"aws_proxy_host"`
	AwsProxyPassword                    string `json:"aws_proxy_password"`
	AwsProxyPort                        uint32 `json:"aws_proxy_port"`
	AwsProxyScheme                      string `json:"aws_proxy_scheme"`
	AwsProxyUsername                    string `json:"aws_proxy_username"`
	AwsRegion                           string `json:"aws_region"`
	AwsSecretAccessKey                  string `json:"aws_secret_access_key"`
	AwsSessionToken                     string `json:"aws_session_token"`
	AwsStsArnRole                       string `json:"aws_sts_arn_role"`
	AwsStsRegion                        string `json:"aws_sts_region"`
	AwsStsSessionName                   string `json:"aws_sts_session_name"`
	AwsStsTimeout                       uint64 `json:"aws_sts_timeout"`
	BpfBufferStorageSize                uint64 `json:"bpf_buffer_storage_size"`
	BpfPerfEventArrayExp                uint64 `json:"bpf_perf_event_array_exp"`
	BufferedLogMax                      uint64 `json:"buffered_log_max"`
	DecorationsTopLevel                 bool   `json:"decorations_top_level"`
	DisableAudit                        bool   `json:"disable_audit"`
	DisableCaching                      bool   `json:"disable_caching"`
	DisableDatabase                     bool   `json:"disable_database"`
	DisableDecorators                   bool   `json:"disable_decorators"`
	DisableDistributed                  bool   `json:"disable_distributed"`
	DisableEvents                       bool   `json:"disable_events"`
	DisableHashCache                    bool   `json:"disable_hash_cache"`
	DisableLogging                      bool   `json:"disable_logging"`
	DisableMemory                       bool   `json:"disable_memory"`
	DistributedDenylistDuration         uint64 `json:"distributed_denylist_duration"`
	DistributedInterval                 uint64 `json:"distributed_interval"`
	DistributedLoginfo                  bool   `json:"distributed_loginfo"`
	DistributedPlugin                   string `json:"distributed_plugin"`
	DistributedTlsMaxAttempts           uint64 `json:"distributed_tls_max_attempts"`
	DistributedTlsReadEndpoint          string `json:"distributed_tls_read_endpoint"`
	DistributedTlsWriteEndpoint         string `json:"distributed_tls_write_endpoint"`
	DockerSocket                        string `json:"docker_socket"`
	EnableBpfEvents                     bool   `json:"enable_bpf_events"`
	EnableFileEvents                    bool   `json:"enable_file_events"`
	EnableForeign                       bool   `json:"enable_foreign"`
	EnableNumericMonitoring             bool   `json:"enable_numeric_monitoring"`
	EnableSyslog                        bool   `json:"enable_syslog"`
	Ephemeral                           bool   `json:"ephemeral"`
	EventsExpiry                        uint64 `json:"events_expiry"`
	EventsMax                           uint64 `json:"events_max"`
	EventsOptimize                      bool   `json:"events_optimize"`
	ExtensionsDefaultIndex              bool   `json:"extensions_default_index"`
	HashCacheMax                        uint32 `json:"hash_cache_max"`
	HostIdentifier                      string `json:"host_identifier"`
	LoggerEventType                     bool   `json:"logger_event_type"`
	LoggerKafkaAcks                     string `json:"logger_kafka_acks"`
	LoggerKafkaBrokers                  string `json:"logger_kafka_brokers"`
	LoggerKafkaCompression              string `json:"logger_kafka_compression"`
	LoggerKafkaTopic                    string `json:"logger_kafka_topic"`
	LoggerMinStatus                     int32  `json:"logger_min_status"`
	LoggerMinStderr                     int32  `json:"logger_min_stderr"`
	LoggerNumerics                      bool   `json:"logger_numerics"`
	LoggerPath                          string `json:"logger_path"`
	LoggerRotate                        bool   `json:"logger_rotate"`
	LoggerRotateMaxFiles                uint64 `json:"logger_rotate_max_files"`
	LoggerRotateSize                    uint64 `json:"logger_rotate_size"`
	LoggerSnapshotEventType             bool   `json:"logger_snapshot_event_type"`
	LoggerSyslogFacility                int32  `json:"logger_syslog_facility"`
	LoggerSyslogPrependCee              bool   `json:"logger_syslog_prepend_cee"`
	LoggerTlsCompress                   bool   `json:"logger_tls_compress"`
	LoggerTlsEndpoint                   string `json:"logger_tls_endpoint"`
	LoggerTlsMaxLines                   uint64 `json:"logger_tls_max_lines"`
	LoggerTlsMaxLinesize                uint64 `json:"logger_tls_max_linesize"`
	LoggerTlsPeriod                     uint64 `json:"logger_tls_period"`
	LxdSocket                           string `json:"lxd_socket"`
	Nullvalue                           string `json:"nullvalue"`
	NumericMonitoringFilesystemPath     string `json:"numeric_monitoring_filesystem_path"`
	NumericMonitoringPlugins            string `json:"numeric_monitoring_plugins"`
	NumericMonitoringPreAggregationTime uint64 `json:"numeric_monitoring_pre_aggregation_time"`
	PackDelimiter                       string `json:"pack_delimiter"`
	PackRefreshInterval                 uint64 `json:"pack_refresh_interval"`
	ReadMax                             uint64 `json:"read_max"`
	ScheduleDefaultInterval             uint64 `json:"schedule_default_interval"`
	ScheduleEpoch                       uint64 `json:"schedule_epoch"`
	ScheduleLognames                    bool   `json:"schedule_lognames"`
	ScheduleMaxDrift                    uint64 `json:"schedule_max_drift"`
	ScheduleReload                      uint64 `json:"schedule_reload"`
	ScheduleSplayPercent                uint64 `json:"schedule_splay_percent"`
	ScheduleTimeout                     uint64 `json:"schedule_timeout"`
	SpecifiedIdentifier                 string `json:"specified_identifier"`
	SyslogEventsExpiry                  uint64 `json:"syslog_events_expiry"`
	SyslogEventsMax                     uint64 `json:"syslog_events_max"`
	SyslogPipePath                      string `json:"syslog_pipe_path"`
	SyslogRateLimit                     uint64 `json:"syslog_rate_limit"`
	TableDelay                          uint64 `json:"table_delay"`
	TableExceptions                     bool   `json:"table_exceptions"`
	ThriftStringSizeLimit               int32  `json:"thrift_string_size_limit"`
	ThriftTimeout                       uint32 `json:"thrift_timeout"`
	ThriftVerbose                       bool   `json:"thrift_verbose"`
	TlsDisableStatusLog                 bool   `json:"tls_disable_status_log"`
	Verbose                             bool   `json:"verbose"`
	WorkerThreads                       int32  `json:"worker_threads"`
	YaraDelay                           uint32 `json:"yara_delay"`

	// embed the os-specific structs
	OsqueryCommandLineFlagsLinux
	OsqueryCommandLineFlagsWindows
	OsqueryCommandLineFlagsMacOS
	OsqueryCommandLineFlagsHidden
}

// NOTE: generate automatically with `go run ./tools/osquery-agent-options/main.go`
type osqueryCommandLineFlags struct {
	AlarmTimeout                        uint64 `json:"alarm_timeout"`
	AuditAllowAcceptSocketEvents        bool   `json:"audit_allow_accept_socket_events"`
	AuditAllowApparmorEvents            bool   `json:"audit_allow_apparmor_events"`
	AuditAllowConfig                    bool   `json:"audit_allow_config"`
	AuditAllowFailedSocketEvents        bool   `json:"audit_allow_failed_socket_events"`
	AuditAllowFimEvents                 bool   `json:"audit_allow_fim_events"`
	AuditAllowForkProcessEvents         bool   `json:"audit_allow_fork_process_events"`
	AuditAllowKillProcessEvents         bool   `json:"audit_allow_kill_process_events"`
	AuditAllowNullAcceptSocketEvents    bool   `json:"audit_allow_null_accept_socket_events"`
	AuditAllowProcessEvents             bool   `json:"audit_allow_process_events"`
	AuditAllowSeccompEvents             bool   `json:"audit_allow_seccomp_events"`
	AuditAllowSelinuxEvents             bool   `json:"audit_allow_selinux_events"`
	AuditAllowSockets                   bool   `json:"audit_allow_sockets"`
	AuditAllowUserEvents                bool   `json:"audit_allow_user_events"`
	AuditBacklogLimit                   int32  `json:"audit_backlog_limit"`
	AuditBacklogWaitTime                int32  `json:"audit_backlog_wait_time"`
	AuditForceReconfigure               bool   `json:"audit_force_reconfigure"`
	AuditForceUnconfigure               bool   `json:"audit_force_unconfigure"`
	AuditPersist                        bool   `json:"audit_persist"`
	AugeasLenses                        string `json:"augeas_lenses"`
	AwsAccessKeyId                      string `json:"aws_access_key_id"`
	AwsDebug                            bool   `json:"aws_debug"`
	AwsEnableProxy                      bool   `json:"aws_enable_proxy"`
	AwsFirehoseEndpoint                 string `json:"aws_firehose_endpoint"`
	AwsFirehosePeriod                   uint64 `json:"aws_firehose_period"`
	AwsFirehoseStream                   string `json:"aws_firehose_stream"`
	AwsKinesisDisableLogStatus          bool   `json:"aws_kinesis_disable_log_status"`
	AwsKinesisEndpoint                  string `json:"aws_kinesis_endpoint"`
	AwsKinesisPeriod                    uint64 `json:"aws_kinesis_period"`
	AwsKinesisRandomPartitionKey        bool   `json:"aws_kinesis_random_partition_key"`
	AwsKinesisStream                    string `json:"aws_kinesis_stream"`
	AwsProfileName                      string `json:"aws_profile_name"`
	AwsProxyHost                        string `json:"aws_proxy_host"`
	AwsProxyPassword                    string `json:"aws_proxy_password"`
	AwsProxyPort                        uint32 `json:"aws_proxy_port"`
	AwsProxyScheme                      string `json:"aws_proxy_scheme"`
	AwsProxyUsername                    string `json:"aws_proxy_username"`
	AwsRegion                           string `json:"aws_region"`
	AwsSecretAccessKey                  string `json:"aws_secret_access_key"`
	AwsSessionToken                     string `json:"aws_session_token"`
	AwsStsArnRole                       string `json:"aws_sts_arn_role"`
	AwsStsRegion                        string `json:"aws_sts_region"`
	AwsStsSessionName                   string `json:"aws_sts_session_name"`
	AwsStsTimeout                       uint64 `json:"aws_sts_timeout"`
	BpfBufferStorageSize                uint64 `json:"bpf_buffer_storage_size"`
	BpfPerfEventArrayExp                uint64 `json:"bpf_perf_event_array_exp"`
	BufferedLogMax                      uint64 `json:"buffered_log_max"`
	CarverBlockSize                     uint32 `json:"carver_block_size"`
	CarverCompression                   bool   `json:"carver_compression"`
	CarverContinueEndpoint              string `json:"carver_continue_endpoint"`
	CarverDisableFunction               bool   `json:"carver_disable_function"`
	CarverExpiry                        uint32 `json:"carver_expiry"`
	CarverStartEndpoint                 string `json:"carver_start_endpoint"`
	ConfigAcceleratedRefresh            uint64 `json:"config_accelerated_refresh"`
	ConfigCheck                         bool   `json:"config_check"`
	ConfigDump                          bool   `json:"config_dump"`
	ConfigEnableBackup                  bool   `json:"config_enable_backup"`
	ConfigPath                          string `json:"config_path"`
	ConfigPlugin                        string `json:"config_plugin"`
	ConfigRefresh                       uint64 `json:"config_refresh"`
	ConfigTlsEndpoint                   string `json:"config_tls_endpoint"`
	ConfigTlsMaxAttempts                uint64 `json:"config_tls_max_attempts"`
	Daemonize                           bool   `json:"daemonize"`
	DatabaseDump                        bool   `json:"database_dump"`
	DatabasePath                        string `json:"database_path"`
	DecorationsTopLevel                 bool   `json:"decorations_top_level"`
	DisableAudit                        bool   `json:"disable_audit"`
	DisableCaching                      bool   `json:"disable_caching"`
	DisableCarver                       bool   `json:"disable_carver"`
	DisableDatabase                     bool   `json:"disable_database"`
	DisableDecorators                   bool   `json:"disable_decorators"`
	DisableDistributed                  bool   `json:"disable_distributed"`
	DisableEnrollment                   bool   `json:"disable_enrollment"`
	DisableEvents                       bool   `json:"disable_events"`
	DisableExtensions                   bool   `json:"disable_extensions"`
	DisableHashCache                    bool   `json:"disable_hash_cache"`
	DisableLogging                      bool   `json:"disable_logging"`
	DisableMemory                       bool   `json:"disable_memory"`
	DisableReenrollment                 bool   `json:"disable_reenrollment"`
	DisableTables                       string `json:"disable_tables"`
	DisableWatchdog                     bool   `json:"disable_watchdog"`
	DistributedDenylistDuration         uint64 `json:"distributed_denylist_duration"`
	DistributedInterval                 uint64 `json:"distributed_interval"`
	DistributedLoginfo                  bool   `json:"distributed_loginfo"`
	DistributedPlugin                   string `json:"distributed_plugin"`
	DistributedTlsMaxAttempts           uint64 `json:"distributed_tls_max_attempts"`
	DistributedTlsReadEndpoint          string `json:"distributed_tls_read_endpoint"`
	DistributedTlsWriteEndpoint         string `json:"distributed_tls_write_endpoint"`
	DockerSocket                        string `json:"docker_socket"`
	EnableBpfEvents                     bool   `json:"enable_bpf_events"`
	EnableExtensionsWatchdog            bool   `json:"enable_extensions_watchdog"`
	EnableFileEvents                    bool   `json:"enable_file_events"`
	EnableForeign                       bool   `json:"enable_foreign"`
	EnableNumericMonitoring             bool   `json:"enable_numeric_monitoring"`
	EnableSyslog                        bool   `json:"enable_syslog"`
	EnableTables                        string `json:"enable_tables"`
	EnrollAlways                        bool   `json:"enroll_always"`
	EnrollSecretEnv                     string `json:"enroll_secret_env"`
	EnrollSecretPath                    string `json:"enroll_secret_path"`
	EnrollTlsEndpoint                   string `json:"enroll_tls_endpoint"`
	Ephemeral                           bool   `json:"ephemeral"`
	EventsExpiry                        uint64 `json:"events_expiry"`
	EventsMax                           uint64 `json:"events_max"`
	EventsOptimize                      bool   `json:"events_optimize"`
	ExtensionsAutoload                  string `json:"extensions_autoload"`
	ExtensionsDefaultIndex              bool   `json:"extensions_default_index"`
	ExtensionsInterval                  uint64 `json:"extensions_interval"`
	ExtensionsRequire                   string `json:"extensions_require"`
	ExtensionsSocket                    string `json:"extensions_socket"`
	ExtensionsTimeout                   uint64 `json:"extensions_timeout"`
	Force                               bool   `json:"force"`
	HashCacheMax                        uint32 `json:"hash_cache_max"`
	HostIdentifier                      string `json:"host_identifier"`
	Install                             bool   `json:"install"`
	KeepContainerWorkerOpen             bool   `json:"keep_container_worker_open"`
	LoggerEventType                     bool   `json:"logger_event_type"`
	LoggerKafkaAcks                     string `json:"logger_kafka_acks"`
	LoggerKafkaBrokers                  string `json:"logger_kafka_brokers"`
	LoggerKafkaCompression              string `json:"logger_kafka_compression"`
	LoggerKafkaTopic                    string `json:"logger_kafka_topic"`
	LoggerMinStatus                     int32  `json:"logger_min_status"`
	LoggerMinStderr                     int32  `json:"logger_min_stderr"`
	LoggerMode                          string `json:"logger_mode"`
	LoggerNumerics                      bool   `json:"logger_numerics"`
	LoggerPath                          string `json:"logger_path"`
	LoggerPlugin                        string `json:"logger_plugin"`
	LoggerRotate                        bool   `json:"logger_rotate"`
	LoggerRotateMaxFiles                uint64 `json:"logger_rotate_max_files"`
	LoggerRotateSize                    uint64 `json:"logger_rotate_size"`
	LoggerSnapshotEventType             bool   `json:"logger_snapshot_event_type"`
	LoggerStderr                        bool   `json:"logger_stderr"`
	LoggerSyslogFacility                int32  `json:"logger_syslog_facility"`
	LoggerSyslogPrependCee              bool   `json:"logger_syslog_prepend_cee"`
	LoggerTlsCompress                   bool   `json:"logger_tls_compress"`
	LoggerTlsEndpoint                   string `json:"logger_tls_endpoint"`
	LoggerTlsMaxLines                   uint64 `json:"logger_tls_max_lines"`
	LoggerTlsMaxLinesize                uint64 `json:"logger_tls_max_linesize"`
	LoggerTlsPeriod                     uint64 `json:"logger_tls_period"`
	Logtostderr                         bool   `json:"logtostderr"`
	LxdSocket                           string `json:"lxd_socket"`
	Nullvalue                           string `json:"nullvalue"`
	NumericMonitoringFilesystemPath     string `json:"numeric_monitoring_filesystem_path"`
	NumericMonitoringPlugins            string `json:"numeric_monitoring_plugins"`
	NumericMonitoringPreAggregationTime uint64 `json:"numeric_monitoring_pre_aggregation_time"`
	PackDelimiter                       string `json:"pack_delimiter"`
	PackRefreshInterval                 uint64 `json:"pack_refresh_interval"`
	Pidfile                             string `json:"pidfile"`
	ProxyHostname                       string `json:"proxy_hostname"`
	ReadMax                             uint64 `json:"read_max"`
	ScheduleDefaultInterval             uint64 `json:"schedule_default_interval"`
	ScheduleEpoch                       uint64 `json:"schedule_epoch"`
	ScheduleLognames                    bool   `json:"schedule_lognames"`
	ScheduleMaxDrift                    uint64 `json:"schedule_max_drift"`
	ScheduleReload                      uint64 `json:"schedule_reload"`
	ScheduleSplayPercent                uint64 `json:"schedule_splay_percent"`
	ScheduleTimeout                     uint64 `json:"schedule_timeout"`
	SpecifiedIdentifier                 string `json:"specified_identifier"`
	Stderrthreshold                     int32  `json:"stderrthreshold"`
	SyslogEventsExpiry                  uint64 `json:"syslog_events_expiry"`
	SyslogEventsMax                     uint64 `json:"syslog_events_max"`
	SyslogPipePath                      string `json:"syslog_pipe_path"`
	SyslogRateLimit                     uint64 `json:"syslog_rate_limit"`
	TableDelay                          uint64 `json:"table_delay"`
	TableExceptions                     bool   `json:"table_exceptions"`
	ThriftStringSizeLimit               int32  `json:"thrift_string_size_limit"`
	ThriftTimeout                       uint32 `json:"thrift_timeout"`
	ThriftVerbose                       bool   `json:"thrift_verbose"`
	TlsClientCert                       string `json:"tls_client_cert"`
	TlsClientKey                        string `json:"tls_client_key"`
	TlsDisableStatusLog                 bool   `json:"tls_disable_status_log"`
	TlsEnrollMaxAttempts                uint64 `json:"tls_enroll_max_attempts"`
	TlsEnrollMaxInterval                uint64 `json:"tls_enroll_max_interval"`
	TlsHostname                         string `json:"tls_hostname"`
	TlsServerCerts                      string `json:"tls_server_certs"`
	TlsSessionReuse                     bool   `json:"tls_session_reuse"`
	TlsSessionTimeout                   uint32 `json:"tls_session_timeout"`
	Uninstall                           bool   `json:"uninstall"`
	Verbose                             bool   `json:"verbose"`
	WatchdogDelay                       uint64 `json:"watchdog_delay"`
	WatchdogForcedShutdownDelay         uint64 `json:"watchdog_forced_shutdown_delay"`
	WatchdogLatencyLimit                uint64 `json:"watchdog_latency_limit"`
	WatchdogLevel                       int32  `json:"watchdog_level"`
	WatchdogMemoryLimit                 uint64 `json:"watchdog_memory_limit"`
	WatchdogUtilizationLimit            uint64 `json:"watchdog_utilization_limit"`
	WorkerThreads                       int32  `json:"worker_threads"`
	YaraDelay                           uint32 `json:"yara_delay"`

	// embed the os-specific structs
	OsqueryCommandLineFlagsLinux
	OsqueryCommandLineFlagsWindows
	OsqueryCommandLineFlagsMacOS
	OsqueryCommandLineFlagsHidden
}

// the following structs are for OS-specific command-line flags supported by
// osquery. They are exported so they can be used by the
// tools/osquery-agent-options script.
type OsqueryCommandLineFlagsLinux struct {
	MallocTrimThreshold   uint64 `json:"malloc_trim_threshold"`
	HardwareDisabledTypes string `json:"hardware_disabled_types"`
}

type OsqueryCommandLineFlagsWindows struct {
	UsersServiceDelay                uint64 `json:"users_service_delay"`
	UsersServiceInterval             uint64 `json:"users_service_interval"`
	GroupsServiceDelay               uint64 `json:"groups_service_delay"`
	GroupsServiceInterval            uint64 `json:"groups_service_interval"`
	EnableNtfsEventPublisher         bool   `json:"enable_ntfs_event_publisher"`
	EnablePowershellEventsSubscriber bool   `json:"enable_powershell_events_subscriber"`
	EnableWindowsEventsPublisher     bool   `json:"enable_windows_events_publisher"`
	EnableWindowsEventsSubscriber    bool   `json:"enable_windows_events_subscriber"`
	NtfsEventPublisherDebug          bool   `json:"ntfs_event_publisher_debug"`
	WindowsEventChannels             string `json:"windows_event_channels"`
	UsnJournalReaderDebug            bool   `json:"usn_journal_reader_debug"`
}

type OsqueryCommandLineFlagsMacOS struct {
	DisableEndpointsecurity    bool   `json:"disable_endpointsecurity"`
	DisableEndpointsecurityFim bool   `json:"disable_endpointsecurity_fim"`
	EnableKeyboardEvents       bool   `json:"enable_keyboard_events"`
	EnableMouseEvents          bool   `json:"enable_mouse_events"`
	EsFimMutePathLiteral       string `json:"es_fim_mute_path_literal"`
	EsFimMutePathPrefix        string `json:"es_fim_mute_path_prefix"`
}

// those osquery flags are not OS-specific, but are also not visible using
// osqueryd --help or select * from osquery_flags, so they can't be generated
// by the osquery-agent-options script.
type OsqueryCommandLineFlagsHidden struct {
	AlsoLogToStderr               bool   `json:"alsologtostderr"`
	EventsStreamingPlugin         string `json:"events_streaming_plugin"`
	LogBufSecs                    int32  `json:"logbufsecs"`
	LogDir                        string `json:"log_dir"`
	MaxLogSize                    int32  `json:"max_log_size"`
	MinLogLevel                   int32  `json:"minloglevel"`
	StopLoggingIfFullDisk         bool   `json:"stop_logging_if_full_disk"`
	AllowUnsafe                   bool   `json:"allow_unsafe"`
	TLSDump                       bool   `json:"tls_dump"`
	AuditDebug                    bool   `json:"audit_debug"`
	AuditFIMDebug                 bool   `json:"audit_fim_debug"`
	AuditShowPartialFIMEvents     bool   `json:"audit_show_partial_fim_events"`
	AuditShowUntrackedResWarnings bool   `json:"audit_show_untracked_res_warnings"`
	AuditFIMShowAccesses          bool   `json:"audit_fim_show_accesses"`
}

// while ValidateJSONAgentOptions validates an entire Agent Options payload,
// this unexported function validates a single set of options. That is, in an
// Agent Options payload, the top-level "config" key defines a set, and each
// of the platform overrides defines other sets. They all have the same
// validation rules.
func validateJSONAgentOptionsSet(rawJSON json.RawMessage) error {
	var opts osqueryAgentOptions
	if err := JSONStrictDecode(bytes.NewReader(rawJSON), &opts); err != nil {
		return err
	}

	// logger TLS endpoint must be a path (starting with "/") if provided
	if opts.Options.LoggerTlsEndpoint != "" {
		if !strings.HasPrefix(opts.Options.LoggerTlsEndpoint, "/") {
			return fmt.Errorf("options.logger_tls_endpoint must be a path starting with '/': %q", opts.Options.LoggerTlsEndpoint)
		}
	}
	return nil
}
