package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

var (
	socket   = flag.String("socket", "", "Path to the extensions UNIX domain socket")
	timeout  = flag.Int("timeout", 3, "Seconds to wait for autoloaded extensions")
	interval = flag.Int("interval", 3, "Seconds delay between connectivity checks")

	// Compiled regex for parsing mdatp health output - handles "key : value" format with variable whitespace
	// Format: "key                                     : value" (key has trailing spaces, colon, then value)
	healthOutputRegex = regexp.MustCompile(`^([^:]+?)\s*:\s*(.+)$`)
)

func main() {
	flag.Parse()
	if *socket == "" {
		log.Fatalln("Missing required --socket argument")
	}

	serverTimeout := osquery.ServerTimeout(
		time.Second * time.Duration(*timeout),
	)
	serverPingInterval := osquery.ServerPingInterval(
		time.Second * time.Duration(*interval),
	)

	server, err := osquery.NewExtensionManagerServer(
		"mdatp_extension",
		*socket,
		serverTimeout,
		serverPingInterval,
	)
	if err != nil {
		log.Fatalf("Error creating extension: %s\n", err)
	}

	server.RegisterPlugin(table.NewPlugin("mdatp_status", mdatpStatusColumns(), generateMDATPStatus))

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}

// mdatpStatusColumns returns the column definitions for the mdatp_status table
func mdatpStatusColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("healthy"),
		table.TextColumn("health_issues"),
		table.TextColumn("licensed"),
		table.TextColumn("engine_version"),
		table.TextColumn("engine_load_status"),
		table.TextColumn("app_version"),
		table.TextColumn("org_id"),
		table.TextColumn("log_level"),
		table.TextColumn("machine_guid"),
		table.TextColumn("release_ring"),
		table.TextColumn("product_expiration"),
		table.TextColumn("cloud_enabled"),
		table.TextColumn("cloud_automatic_sample_submission_consent"),
		table.TextColumn("cloud_diagnostic_enabled"),
		table.TextColumn("cloud_pin_certificate_thumbs"),
		table.TextColumn("passive_mode_enabled"),
		table.TextColumn("behavior_monitoring"),
		table.TextColumn("real_time_protection_enabled"),
		table.TextColumn("real_time_protection_available"),
		table.TextColumn("real_time_protection_subsystem"),
		table.TextColumn("network_events_subsystem"),
		table.TextColumn("device_control_enforcement_level"),
		table.TextColumn("tamper_protection"),
		table.TextColumn("automatic_definition_update_enabled"),
		table.TextColumn("definitions_updated"),
		table.TextColumn("definitions_updated_minutes_ago"),
		table.TextColumn("definitions_version"),
		table.TextColumn("definitions_status"),
		table.TextColumn("edr_early_preview_enabled"),
		table.TextColumn("edr_device_tags"),
		table.TextColumn("edr_group_ids"),
		table.TextColumn("edr_configuration_version"),
		table.TextColumn("edr_machine_id"),
		table.TextColumn("conflicting_applications"),
		table.TextColumn("network_protection_status"),
		table.TextColumn("network_protection_enforcement_level"),
		table.TextColumn("data_loss_prevention_status"),
		table.TextColumn("full_disk_access_enabled"),
		table.TextColumn("troubleshooting_mode"),
		table.TextColumn("ecs_configuration_ids"),
		table.TextColumn("error"),
	}
}

// generateMDATPStatus generates rows for the mdatp_status table
func generateMDATPStatus(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	mdatpPath, err := findMDATPPath()
	if err != nil {
		return []map[string]string{
			{"error": err.Error()},
		}, nil
	}

	output, err := executeMDATPCommand(mdatpPath)
	result := parseMDATPHealthOutput(output)

	if err != nil {
		result["error"] = err.Error()
	}

	return []map[string]string{result}, nil
}

// findMDATPPath locates the mdatp binary by checking common installation paths
func findMDATPPath() (string, error) {
	possiblePaths := []string{
		"/usr/local/bin/mdatp",
		"/opt/microsoft/mdatp/bin/mdatp",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("mdatp binary not found at %s or %s", possiblePaths[0], possiblePaths[1])
}

// executeMDATPCommand calls mdatp with the health flag
func executeMDATPCommand(mdatpPath string) (string, error) {
	cmd := exec.Command(mdatpPath, "health")
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// parseMDATPHealthOutput parses the output from 'mdatp health'
func parseMDATPHealthOutput(output string) map[string]string {
	result := make(map[string]string)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "ATTENTION:") {
			continue
		}

		matches := healthOutputRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			// Extract and trim the key (removes trailing spaces from keys like "healthy                                     ")
			key := strings.TrimSpace(matches[1])
			// Extract and trim the value
			value := strings.TrimSpace(matches[2])

			// Remove surrounding quotes if present - cleans up the values returned
			if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
				value = value[1 : len(value)-1]
			}

			// Convert key to lowercase and replace spaces with underscores - standardizes the keys
			key = strings.ToLower(strings.ReplaceAll(key, " ", "_"))

			// Only store if we successfully extracted both key and value
			if key != "" && value != "" {
				result[key] = value
			}
		}
	}

	return result
}
