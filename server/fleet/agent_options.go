package fleet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type AgentOptions struct {
	// Config is the base config options.
	Config json.RawMessage `json:"config"`
	// Overrides includes any platform-based overrides.
	Overrides AgentOptionsOverrides `json:"overrides,omitempty"`
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
func ValidateJSONAgentOptions(rawJSON json.RawMessage) error {
	var opts AgentOptions
	if err := jsonStrictDecode(rawJSON, &opts); err != nil {
		return err
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
	return nil
}

// JSON definition of the available configuration options in osquery.
// See https://osquery.readthedocs.io/en/stable/deployment/configuration/#configuration-specification
//
// NOTE: Update the following line with the version used for validation.
//   Current version: 5.4.0
type osqueryAgentOptions struct {
	Options osqueryOptions `json:"options"`

	Schedule map[string]struct {
		Query    string `json:"query"`
		Interval int    `json:"interval"`
		Removed  bool   `json:"removed"`
		Snapshot bool   `json:"snapshot"`
		Platform string `json:"platform"`
		Version  string `json:"version"`
		Shard    int    `json:"shard"`
		Denylist bool   `json:"denylist"`
	} `json:"schedule"`

	// Packs may have a string or struct value, both are supported, so a raw value
	// is used to unmarshal and the type-check is done after. When it is a struct,
	// it must be compatible with the osqueryPack struct below.
	Packs map[string]json.RawMessage `json:"packs"`

	FilePaths    map[string][]string `json:"file_paths"`
	FileAccesses []string            `json:"file_accesses"`

	YARA struct {
		Signatures map[string][]string `json:"signatures"`
		FilePaths  map[string][]string `json:"file_paths"`
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

// When the osqueryAgentOptions.Packs field maps to a struct, this is the
// definition that the struct must have.
type osqueryPack struct {
	Queries map[string]struct {
		Query       string `json:"query"`
		Interval    int    `json:"interval"`
		Description string `json:"description"`
		// TODO(mna): unclear if the following fields can be present in a pack's query?
		Removed  bool `json:"removed"`
		Snapshot bool `json:"snapshot"`
		Denylist bool `json:"denylist"`
	} `json:"schedule"`
	Platform  string   `json:"platform"`
	Version   string   `json:"version"`
	Shard     int      `json:"shard"`
	Discovery []string `json:"discovery"`
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
	HardwareDisabledTypes               string `json:"hardware_disabled_types"`
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
	YaraMallocTrim                      bool   `json:"yara_malloc_trim"`
}

// while ValidateJSONAgentOptions validates an entire Agent Options payload,
// this unexported function validates a single set of options. That is, in an
// Agent Options payload, the top-level "config" key defines a set, and each
// of the platform overrides defines other sets. They all have the same
// validation rules.
func validateJSONAgentOptionsSet(rawJSON json.RawMessage) error {
	var opts osqueryAgentOptions
	if err := jsonStrictDecode(rawJSON, &opts); err != nil {
		return err
	}

	// Packs may have a string or struct value, both are supported
	for packKey, pack := range opts.Packs {
		if len(pack) == 0 {
			// should never happen, just to make sure we avoid a panic reading the first byte
			continue
		}
		switch pack[0] {
		case '"':
			// a string, this is fine
		case '{':
			// an object, must match the pack struct
			var packStruct osqueryPack
			if err := jsonStrictDecode(pack, &packStruct); err != nil {
				return fmt.Errorf("pack %q: %w", packKey, err)
			}
		case 't', 'f':
			return fmt.Errorf("pack %q: invalid bool value, expected string or object", packKey)
		case 'n':
			return fmt.Errorf("pack %q: invalid null value, expected string or object", packKey)
		case '[':
			return fmt.Errorf("pack %q: invalid array value, expected string or object", packKey)
		default:
			return fmt.Errorf("pack %q: invalid number value, expected string or object", packKey)
		}
	}

	return nil
}

func jsonStrictDecode(rawJSON json.RawMessage, v interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(rawJSON))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}

	var extra json.RawMessage
	if dec.Decode(&extra) != io.EOF {
		return errors.New("json: extra bytes after end of object")
	}

	return nil
}
