package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

//go:generate go run ../../tools/osquery-agent-options agent_options_generated.go

const maxAgentScriptExecutionTimeout = 3600

type AgentOptions struct {
	// ScriptExecutionTimeout is the maximum time in seconds that a script can run.
	ScriptExecutionTimeout int `json:"script_execution_timeout,omitempty"`
	// Config is the base config options.
	Config json.RawMessage `json:"config"`
	// Overrides includes any platform-based overrides.
	Overrides AgentOptionsOverrides `json:"overrides,omitempty"`
	// CommandLineStartUpFlags are the osquery CLI_FLAGS
	CommandLineStartUpFlags json.RawMessage `json:"command_line_flags,omitempty"`
	// Extensions are the orbit managed extensions
	Extensions json.RawMessage `json:"extensions,omitempty"`
	// UpdateChannels holds the configured channels for fleetd components.
	UpdateChannels json.RawMessage `json:"update_channels,omitempty"`
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
	err := validateJSONAgentOptionsInternal(ctx, ds, rawJSON, isPremium)
	if field := GetJSONUnknownField(err); field != nil {
		correctKeyPath, keyErr := findAgentOptionsKeyPath(*field)
		if keyErr != nil {
			return fmt.Errorf("agent options struct parsing: %w", err)
		}
		var keyPathJoined string
		switch pathLen := len(correctKeyPath); {
		case pathLen > 1:
			keyPathJoined = fmt.Sprintf("%q", strings.Join(correctKeyPath[:len(correctKeyPath)-1], "."))
		case pathLen == 1:
			keyPathJoined = "top level"
		}
		if keyPathJoined != "" {
			err = fmt.Errorf("%q should be part of the %s object", *field, keyPathJoined)
		}
	}
	return err
}

func validateJSONAgentOptionsInternal(ctx context.Context, ds Datastore, rawJSON json.RawMessage, isPremium bool) error {
	var opts AgentOptions
	if err := JSONStrictDecode(bytes.NewReader(rawJSON), &opts); err != nil {
		return err
	}

	if opts.ScriptExecutionTimeout > maxAgentScriptExecutionTimeout {
		return fmt.Errorf("'script_execution_timeout' value exceeds limit. Maximum value is %d", maxAgentScriptExecutionTimeout)
	}

	if len(opts.CommandLineStartUpFlags) > 0 {
		var flags osqueryCommandLineFlags
		if err := JSONStrictDecode(bytes.NewReader(opts.CommandLineStartUpFlags), &flags); err != nil {
			return fmt.Errorf("command-line flags: %w", err)
		}

		// We prevent setting the following flags because they can break fleetd.
		flagNotSupportedErr := "The %s flag isn't supported. Please remove this flag."
		if flags.HostIdentifier != "" {
			return fmt.Errorf(flagNotSupportedErr, "--host_identifier")
		}
		if flags.ExtensionsAutoload != "" {
			return fmt.Errorf(flagNotSupportedErr, "--extensions_autoload")
		}
		if flags.DatabasePath != "" {
			return fmt.Errorf(flagNotSupportedErr, "--database_path")
		}
	}

	if len(opts.UpdateChannels) > 0 {
		if !isPremium {
			// The update_channels feature is premium only.
			return ErrMissingLicense
		}
		if string(opts.UpdateChannels) == "null" {
			return errors.New("update_channels cannot be null")
		}
		if err := checkEmptyFields("update_channels", opts.UpdateChannels); err != nil {
			return err
		}
		var updateChannels OrbitUpdateChannels
		if err := JSONStrictDecode(bytes.NewReader(opts.UpdateChannels), &updateChannels); err != nil {
			return fmt.Errorf("update_channels: %w", err)
		}
	}

	if len(opts.Config) > 0 {
		if err := validateJSONAgentOptionsSet(opts.Config); err != nil {
			return fmt.Errorf("common config: %w", err)
		}
	}

	for platform, platformOpts := range opts.Overrides.Platforms {
		if len(platformOpts) > 0 {
			if string(platformOpts) == "null" {
				return errors.New("platforms cannot be null. To remove platform overrides omit overrides from agent options.")
			}

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

func checkEmptyFields(prefix string, data json.RawMessage) error {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("unmarshal data: %w", err)
	}
	for k, v := range m {
		if v == nil {
			return fmt.Errorf("%s.%s is defined but not set", prefix, k)
		}
		if s, ok := v.(string); ok && s == "" {
			return fmt.Errorf("%s.%s is set to an empty string", prefix, k)
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

// the following structs are for OS-specific command-line flags supported by
// osquery. They are exported so they can be used by the
// tools/osquery-agent-options script.
type OsqueryCommandLineFlagsLinux struct {
	AuditAllowAcceptSocketEvents             bool   `json:"audit_allow_accept_socket_events"`
	AuditAllowApparmorEvents                 bool   `json:"audit_allow_apparmor_events"`
	AuditAllowFailedSocketEvents             bool   `json:"audit_allow_failed_socket_events"`
	AuditAllowForkProcessEvents              bool   `json:"audit_allow_fork_process_events"`
	AuditAllowKillProcessEvents              bool   `json:"audit_allow_kill_process_events"`
	AuditAllowNullAcceptSocketEvents         bool   `json:"audit_allow_null_accept_socket_events"`
	AuditAllowSeccompEvents                  bool   `json:"audit_allow_seccomp_events"`
	AuditAllowSelinuxEvents                  bool   `json:"audit_allow_selinux_events"`
	AuditBacklogLimit                        int32  `json:"audit_backlog_limit"`
	AuditBacklogWaitTime                     int32  `json:"audit_backlog_wait_time"`
	AuditForceReconfigure                    bool   `json:"audit_force_reconfigure"`
	AuditForceUnconfigure                    bool   `json:"audit_force_unconfigure"`
	AuditPersist                             bool   `json:"audit_persist"`
	BpfBufferStorageSize                     uint64 `json:"bpf_buffer_storage_size"`
	BpfPerfEventArrayExp                     uint64 `json:"bpf_perf_event_array_exp"`
	DisableMemory                            bool   `json:"disable_memory"`
	EnableBpfEvents                          bool   `json:"enable_bpf_events"`
	EnableSyslog                             bool   `json:"enable_syslog"`
	ExperimentsLinuxeventsCircularBufferSize uint32 `json:"experiments_linuxevents_circular_buffer_size"`
	ExperimentsLinuxeventsPerfOutputSize     uint32 `json:"experiments_linuxevents_perf_output_size"`
	HardwareDisabledTypes                    string `json:"hardware_disabled_types"`
	KeepContainerWorkerOpen                  bool   `json:"keep_container_worker_open"`
	LxdSocket                                string `json:"lxd_socket"`
	MallocTrimThreshold                      uint64 `json:"malloc_trim_threshold"`
	SyslogEventsExpiry                       uint64 `json:"syslog_events_expiry"`
	SyslogEventsMax                          uint64 `json:"syslog_events_max"`
	SyslogPipePath                           string `json:"syslog_pipe_path"`
	SyslogRateLimit                          uint64 `json:"syslog_rate_limit"`
}

type OsqueryCommandLineFlagsWindows struct {
	UsersServiceDelay                uint64 `json:"users_service_delay"`
	UsersServiceInterval             uint64 `json:"users_service_interval"`
	GroupsServiceDelay               uint64 `json:"groups_service_delay"`
	GroupsServiceInterval            uint64 `json:"groups_service_interval"`
	EnableNtfsEventPublisher         bool   `json:"enable_ntfs_event_publisher"`
	EnablePowershellEventsSubscriber bool   `json:"enable_powershell_events_subscriber"`
	EnableProcessEtwEvents           bool   `json:"enable_process_etw_events"`
	EnableWindowsEventsPublisher     bool   `json:"enable_windows_events_publisher"`
	EnableWindowsEventsSubscriber    bool   `json:"enable_windows_events_subscriber"`
	EtwKernelTraceBufferSize         uint32 `json:"etw_kernel_trace_buffer_size"`
	EtwKernelTraceFlushTimer         uint32 `json:"etw_kernel_trace_flush_timer"`
	EtwKernelTraceMaximumBuffers     uint32 `json:"etw_kernel_trace_maximum_buffers"`
	EtwKernelTraceMinimumBuffers     uint32 `json:"etw_kernel_trace_minimum_buffers"`
	EtwUserspaceTraceBufferSize      uint32 `json:"etw_userspace_trace_buffer_size"`
	EtwUserspaceTraceFlushTimer      uint32 `json:"etw_userspace_trace_flush_timer"`
	EtwUserspaceTraceMaximumBuffers  uint32 `json:"etw_userspace_trace_maximum_buffers"`
	EtwUserspaceTraceMinimumBuffers  uint32 `json:"etw_userspace_trace_minimum_buffers"`
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
	IgnoreRegistryExceptions      bool   `json:"ignore_registry_exceptions"`
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

func findAgentOptionsKeyPath(key string) ([]string, error) {
	if key == "script_execution_timeout" {
		return []string{"script_execution_timeout"}, nil
	}

	configPath, err := locateStructJSONKeyPath(key, "config", osqueryAgentOptions{})
	if err != nil {
		return nil, fmt.Errorf("locating key path in agent options: %w", err)
	}
	if configPath != nil {
		return configPath, nil
	}

	if key == "overrides" {
		return []string{"overrides"}, nil
	}
	if key == "platforms" {
		return []string{"overrides", "platforms"}, nil
	}

	commandLinePath, err := locateStructJSONKeyPath(key, "command_line_flags", osqueryCommandLineFlags{})
	if err != nil {
		return nil, fmt.Errorf("locating key path in agent command line options: %w", err)
	}
	if commandLinePath != nil {
		return commandLinePath, nil
	}

	extensionsPath, err := locateStructJSONKeyPath(key, "extensions", ExtensionInfo{})
	if err != nil {
		return nil, fmt.Errorf("locating key path in agent extensions options: %w", err)
	}
	if extensionsPath != nil {
		return extensionsPath, nil
	}

	channelsPath, err := locateStructJSONKeyPath(key, "update_channels", OrbitUpdateChannels{})
	if err != nil {
		return nil, fmt.Errorf("locating key path in agent update channels: %w", err)
	}
	if channelsPath != nil {
		return channelsPath, nil
	}

	return nil, nil
}

// Only searches two layers deep
func locateStructJSONKeyPath(key, startKey string, target any) ([]string, error) {
	if key == startKey {
		return []string{startKey}, nil
	}

	optionsBytes, err := json.Marshal(target)
	if err != nil {
		return nil, fmt.Errorf("unable to marshall target: %w", err)
	}

	var opts map[string]any

	if err := json.Unmarshal(optionsBytes, &opts); err != nil {
		return nil, fmt.Errorf("unable to unmarshall target: %w", err)
	}

	var path [3]string
	path[0] = startKey
	for k, v := range opts {
		path[1] = k
		if k == key {
			return path[:2], nil
		}

		switch v.(type) {
		case map[string]any:
			for k2 := range v.(map[string]any) {
				path[2] = k2
				if key == k2 {
					return path[:3], nil
				}
			}
		}
	}

	return nil, nil
}
