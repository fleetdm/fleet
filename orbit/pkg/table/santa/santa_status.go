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
	WatchItems struct {
		Enabled bool `json:"enabled"`
	} `json:"watch_items"`
	Daemon struct {
		FileLogging       bool    `json:"file_logging"`
		WatchdogRamEvents int     `json:"watchdog_ram_events"`
		DriverConnected   bool    `json:"driver_connected"`
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
		Enabled            bool   `json:"enabled"`
		LastSuccessfulRule string `json:"last_successful_rule"`
		PushNotifications  string `json:"push_notifications"`
		BundleScanning     bool   `json:"bundle_scanning"`
		CleanRequired      bool   `json:"clean_required"`
		Server             string `json:"server"`
		LastSuccessfulFull string `json:"last_successful_full"`
	} `json:"sync"`
	Metrics struct {
		Enabled bool `json:"enabled"`
	} `json:"metrics"`
}

func StatusColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("last_successful_rule"),
		table.TextColumn("push_notifications"),
		table.IntegerColumn("bundle_scanning"),
		table.IntegerColumn("clean_required"),
		table.TextColumn("server"),
		table.TextColumn("last_successful_full"),
		table.IntegerColumn("file_logging"),
		table.IntegerColumn("watchdog_ram_events"),
		table.IntegerColumn("driver_connected"),
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
		"last_successful_rule":     status.Sync.LastSuccessfulRule,
		"push_notifications":       status.Sync.PushNotifications,
		"bundle_scanning":          boolToIntString(status.Sync.BundleScanning),
		"clean_required":           boolToIntString(status.Sync.CleanRequired),
		"server":                   status.Sync.Server,
		"last_successful_full":     status.Sync.LastSuccessfulFull,
		"file_logging":             boolToIntString(status.Daemon.FileLogging),
		"watchdog_ram_events":      strconv.Itoa(status.Daemon.WatchdogRamEvents),
		"driver_connected":         boolToIntString(status.Daemon.DriverConnected),
		"log_type":                 status.Daemon.LogType,
		"watchdog_cpu_events":      strconv.Itoa(status.Daemon.WatchdogCpuEvents),
		"mode":                     status.Daemon.Mode,
		"watchdog_cpu_peak":        floatToString(status.Daemon.WatchdogCpuPeak),
		"watchdog_ram_peak":        floatToString(status.Daemon.WatchdogRamPeak),
		"transitive_rules_enabled": boolToIntString(status.TransitiveAllowlisting.Enabled),
		"remount_usb_mode":         status.Daemon.RemountUsbMode,
		"block_usb":                boolToIntString(status.Daemon.BlockUsb),
		"on_start_usb_options":     status.Daemon.OnStartUsbOptions,
		"root_cache_count":         strconv.Itoa(status.Cache.RootCacheCount),
		"non_root_cache_count":     strconv.Itoa(status.Cache.NonRootCacheCount),
		"static_rule_count":        strconv.Itoa(status.Daemon.StaticRules),
		"certificate_rules":        strconv.Itoa(status.RuleTypes.CertificateRules),
		"cdhash_rules":             strconv.Itoa(status.RuleTypes.CdhashRules),
		"transitive_rules_count":   strconv.Itoa(status.TransitiveAllowlisting.TransitiveRules),
		"teamid_rules":             strconv.Itoa(status.RuleTypes.TeamidRules),
		"signingid_rules":          strconv.Itoa(status.RuleTypes.SigningidRules),
		"compiler_rules":           strconv.Itoa(status.TransitiveAllowlisting.CompilerRules),
		"binary_rules":             strconv.Itoa(status.RuleTypes.BinaryRules),
		"events_pending_upload":    "0",
		"watch_items_enabled":      boolToIntString(status.WatchItems.Enabled),
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
