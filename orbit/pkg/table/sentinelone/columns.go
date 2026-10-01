package sentinelone

// columnOrder is the column order reported by Columns. Every entry must appear
// as a value in columnPathMap (enforced by TestColumnMapCoversColumnOrder).
var columnOrder = []string{
	// Agent
	"agent_version",
	"agent_id",
	"install_date",
	"es_framework",
	"operational_state",
	"remote_profiler",
	"network_monitoring",
	"network_extension",
	"network_extension_content_filter",
	"ready",
	"protection",
	"infected",
	"network_quarantine",
	"compatible_os",
	// Command
	"command_authentication",
	// Daemons -> Services
	"service_agent_helper",
	"service_agent_ui",
	"service_cleaner",
	"service_control_service",
	"service_framework",
	"service_guard",
	"service_helper_service",
	"service_lib_hooks_service",
	"service_lib_logs_service",
	"service_shell",
	// Daemons -> Integrity
	"integrity_sentineld",
	"integrity_sentineld_guard",
	"integrity_sentineld_helper",
	"integrity_sentineld_shell",
	// Launchd
	"launchd_agent_helper",
	"launchd_agent_ui",
	"launchd_sentinel_extensions",
	"launchd_sentineld",
	"launchd_sentineld_guard",
	"launchd_sentineld_helper",
	"launchd_sentineld_shell",
	// Management
	"management_server",
	"management_site_key",
	"management_last_seen",
	"management_connected",
	// Windows-only
	"tamper_protection",
	"agent_run_mode",
	"mitigation_policy",
}

// columnPathMap maps the underscore-joined section path that parseStatus
// produces to the column it populates. A path that isn't listed here is
// ignored, so fields added by a future SentinelOne release are dropped rather
// than breaking the table.
var columnPathMap = map[string]string{
	// Agent section.
	"agent_version":                          "agent_version",
	"agent_id":                               "agent_id",
	"agent_install_date":                     "install_date",
	"agent_es_framework":                     "es_framework",
	"agent_agent_operational_state":          "operational_state",
	"agent_remote_profiler":                  "remote_profiler",
	"agent_agent_network_monitoring":         "network_monitoring",
	"agent_network_extension":                "network_extension",
	"agent_network_extension_content_filter": "network_extension_content_filter",
	"agent_ready":                            "ready",
	"agent_protection":                       "protection",
	"agent_infected":                         "infected",
	"agent_network_quarantine":               "network_quarantine",
	"agent_compatible_os":                    "compatible_os",

	// Command section.
	"command_authentication": "command_authentication",

	// Daemons -> Services.
	"daemons_services_agent_helper":      "service_agent_helper",
	"daemons_services_agent_ui":          "service_agent_ui",
	"daemons_services_cleaner":           "service_cleaner",
	"daemons_services_control_service":   "service_control_service",
	"daemons_services_framework":         "service_framework",
	"daemons_services_guard":             "service_guard",
	"daemons_services_helper_service":    "service_helper_service",
	"daemons_services_lib_hooks_service": "service_lib_hooks_service",
	"daemons_services_lib_logs_service":  "service_lib_logs_service",
	"daemons_services_shell":             "service_shell",

	// Daemons -> Integrity.
	"daemons_integrity_sentineld":        "integrity_sentineld",
	"daemons_integrity_sentineld_guard":  "integrity_sentineld_guard",
	"daemons_integrity_sentineld_helper": "integrity_sentineld_helper",
	"daemons_integrity_sentineld_shell":  "integrity_sentineld_shell",

	// Launchd section.
	"launchd_agent_helper":        "launchd_agent_helper",
	"launchd_agent_ui":            "launchd_agent_ui",
	"launchd_sentinel_extensions": "launchd_sentinel_extensions",
	"launchd_sentineld":           "launchd_sentineld",
	"launchd_sentineld_guard":     "launchd_sentineld_guard",
	"launchd_sentineld_helper":    "launchd_sentineld_helper",
	"launchd_sentineld_shell":     "launchd_sentineld_shell",

	// Management section.
	"management_server":    "management_server",
	"management_site_key":  "management_site_key",
	"management_last_seen": "management_last_seen",
	"management_connected": "management_connected",

	// Windows-only fields with no macOS/Linux equivalent.
	// "Self-Protection status: On", normalized by applyWindowsCanonicalPaths.
	"tamper_protection": "tamper_protection",
	// "SentinelAgent is running as PPL" / "Mitigation policy: none".
	"sentinelagent_is_running": "agent_run_mode",
	"mitigation_policy":        "mitigation_policy",
}

// epochColumns are the columns holding timestamps, reported as Unix epoch
// seconds.
var epochColumns = []string{
	"install_date",
	"management_last_seen",
}
