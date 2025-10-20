//go:build darwin

// Package santa implements the tables for getting Santa data
// (logs/status) on macOS.
//
// Santa is an open source macOS endpoint security system with
// binary whitelisting and blacklisting capabilities.
// Based on https://github.com/allenhouchins/fleet-extensions/tree/main/santa
package santa

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// declare execCommandContext for testing
var execCommandContext = exec.CommandContext

type santaStatus struct {
	Daemon struct {
		FileLogging       bool    `json:"file_logging"`
		WatchdogRamEvents int     `json:"watchdog_ram_events"`
		LogType           string  `json:"log_type"`
		WatchdogCpuEvents int     `json:"watchdog_cpu_events"`
		Mode              string  `json:"mode"`
		WatchdogCpuPeak   float64 `json:"watchdog_cpu_peak"`
		WatchdogRamPeak   float64 `json:"watchdog_ram_peak"`
		StaticRules       int     `json:"static_rules"`
		RemountUsbMode    string  `json:"remount_usb_mode"`
		BlockUsb          bool    `json:"block_usb"`
		OnStartUsbOptions string  `json:"on_start_usb_options"`
	} `json:"daemon"`
	Cache struct {
		RootCacheCount    int `json:"root_cache_count"`
		NonRootCacheCount int `json:"non_root_cache_count"`
	} `json:"cache"`
	RuleTypes struct {
		CertificateRules int `json:"certificate_rules"`
		CdhashRules      int `json:"cdhash_rules"`
		TeamidRules      int `json:"teamid_rules"`
		SigningidRules   int `json:"signingid_rules"`
		BinaryRules      int `json:"binary_rules"`
	} `json:"rule_types"`
	TransitiveAllowlisting struct {
		Enabled         bool `json:"enabled"`
		CompilerRules   int  `json:"compiler_rules"`
		TransitiveRules int  `json:"transitive_rules"`
	} `json:"transitive_allowlisting"`
	Sync struct {
		Enabled             bool   `json:"enabled"`
		Server              string `json:"server"`
		CleanRequired       bool   `json:"clean_required"`
		LastSuccessfulFull  string `json:"last_successful_full"`
		LastSuccessfulRule  string `json:"last_successful_rule"`
		PushNotifications   string `json:"push_notifications"`
		BundleScanning      bool   `json:"bundle_scanning"`
		EventsPendingUpload int    `json:"events_pending_upload"`
		ExecutionRulesHash  string `json:"execution_rules_hash"`
		FileAccessRulesHash string `json:"file_access_rules_hash"`
	} `json:"sync"`
	WatchItems struct {
		Enabled          bool   `json:"enabled"`
		DataSource       string `json:"data_source"`
		RuleCount        int    `json:"rule_count"`
		LastPolicyUpdate string `json:"last_policy_update"`
		PolicyVersion    string `json:"policy_version"`
		ConfigPath       string `json:"config_path"`
	} `json:"watch_items"`
	Metrics struct {
		Enabled               bool   `json:"enabled"`
		Server                string `json:"server"`
		ExportIntervalSeconds int    `json:"export_interval_seconds"`
	} `json:"metrics"`
}

func StatusColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.IntegerColumn("file_logging"),
		table.IntegerColumn("watchdog_ram_events"),
		table.TextColumn("log_type"),
		table.IntegerColumn("watchdog_cpu_events"),
		table.TextColumn("mode"),
		table.DoubleColumn("watchdog_cpu_peak"),
		table.DoubleColumn("watchdog_ram_peak"),
		table.IntegerColumn("transitive_rules_enabled"),
		table.TextColumn("remount_usb_mode"),
		table.IntegerColumn("block_usb"),
		table.TextColumn("on_start_usb_options"),
		table.IntegerColumn("root_cache_count"),
		table.IntegerColumn("non_root_cache_count"),
		table.IntegerColumn("static_rule_count"),
		table.IntegerColumn("certificate_rules"),
		table.IntegerColumn("cdhash_rules"),
		table.IntegerColumn("transitive_rules_count"),
		table.IntegerColumn("teamid_rules"),
		table.IntegerColumn("signingid_rules"),
		table.IntegerColumn("compiler_rules"),
		table.IntegerColumn("binary_rules"),
		table.IntegerColumn("events_pending_upload"),
		table.IntegerColumn("watch_items_enabled"),
		table.IntegerColumn("sync_enabled"),
		table.TextColumn("sync_server"),
		table.IntegerColumn("sync_clean_required"),
		table.TextColumn("sync_last_successful_full"),
		table.TextColumn("sync_last_successful_rule"),
		table.TextColumn("sync_push_notifications"),
		table.IntegerColumn("sync_bundle_scanning"),
		table.TextColumn("sync_execution_rules_hash"),
		table.TextColumn("sync_file_access_rules_hash"),
		table.TextColumn("watch_items_data_source"),
		table.IntegerColumn("watch_items_rule_count"),
		table.TextColumn("watch_items_last_policy_update"),
		table.TextColumn("watch_items_policy_version"),
		table.TextColumn("watch_items_config_path"),
		table.IntegerColumn("metrics_enabled"),
		table.TextColumn("metrics_server"),
		table.IntegerColumn("metrics_export_interval_seconds"),
	}
}

func GenerateStatus(ctx context.Context, _ table.QueryContext) ([]map[string]string, error) {
	cmd := execCommandContext(ctx, "/usr/local/bin/santactl", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		// Gracefully return an empty result if santactl fails
		log.Debug().Err(err).Msg("failed to run santactl status --json")
		return []map[string]string{}, nil
	}

	var status santaStatus
	if err := json.Unmarshal(output, &status); err != nil {
		return nil, err
	}

	row := map[string]string{
		"file_logging":                    boolToIntString(status.Daemon.FileLogging),
		"watchdog_ram_events":             strconv.Itoa(status.Daemon.WatchdogRamEvents),
		"log_type":                        status.Daemon.LogType,
		"watchdog_cpu_events":             strconv.Itoa(status.Daemon.WatchdogCpuEvents),
		"mode":                            status.Daemon.Mode,
		"watchdog_cpu_peak":               floatToString(status.Daemon.WatchdogCpuPeak),
		"watchdog_ram_peak":               floatToString(status.Daemon.WatchdogRamPeak),
		"transitive_rules_enabled":        boolToIntString(status.TransitiveAllowlisting.Enabled),
		"remount_usb_mode":                status.Daemon.RemountUsbMode,
		"block_usb":                       boolToIntString(status.Daemon.BlockUsb),
		"on_start_usb_options":            status.Daemon.OnStartUsbOptions,
		"root_cache_count":                strconv.Itoa(status.Cache.RootCacheCount),
		"non_root_cache_count":            strconv.Itoa(status.Cache.NonRootCacheCount),
		"static_rule_count":               strconv.Itoa(status.Daemon.StaticRules),
		"certificate_rules":               strconv.Itoa(status.RuleTypes.CertificateRules),
		"cdhash_rules":                    strconv.Itoa(status.RuleTypes.CdhashRules),
		"transitive_rules_count":          strconv.Itoa(status.TransitiveAllowlisting.TransitiveRules),
		"teamid_rules":                    strconv.Itoa(status.RuleTypes.TeamidRules),
		"signingid_rules":                 strconv.Itoa(status.RuleTypes.SigningidRules),
		"compiler_rules":                  strconv.Itoa(status.TransitiveAllowlisting.CompilerRules),
		"binary_rules":                    strconv.Itoa(status.RuleTypes.BinaryRules),
		"events_pending_upload":           strconv.Itoa(status.Sync.EventsPendingUpload),
		"watch_items_enabled":             boolToIntString(status.WatchItems.Enabled),
		"sync_enabled":                    boolToIntString(status.Sync.Enabled),
		"sync_server":                     status.Sync.Server,
		"sync_clean_required":             boolToIntString(status.Sync.CleanRequired),
		"sync_last_successful_full":       status.Sync.LastSuccessfulFull,
		"sync_last_successful_rule":       status.Sync.LastSuccessfulRule,
		"sync_push_notifications":         status.Sync.PushNotifications,
		"sync_bundle_scanning":            boolToIntString(status.Sync.BundleScanning),
		"sync_execution_rules_hash":       status.Sync.ExecutionRulesHash,
		"sync_file_access_rules_hash":     status.Sync.FileAccessRulesHash,
		"watch_items_data_source":         status.WatchItems.DataSource,
		"watch_items_rule_count":          strconv.Itoa(status.WatchItems.RuleCount),
		"watch_items_last_policy_update":  status.WatchItems.LastPolicyUpdate,
		"watch_items_policy_version":      status.WatchItems.PolicyVersion,
		"watch_items_config_path":         status.WatchItems.ConfigPath,
		"metrics_enabled":                 boolToIntString(status.Metrics.Enabled),
		"metrics_server":                  status.Metrics.Server,
		"metrics_export_interval_seconds": strconv.Itoa(status.Metrics.ExportIntervalSeconds),
	}

	return []map[string]string{row}, nil
}

func boolToIntString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func floatToString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
