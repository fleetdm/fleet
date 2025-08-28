package tables

import (
	"database/sql"
	"encoding/json"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20250828120836, Down_20250828120836)
}

func Up_20250828120836(tx *sql.Tx) error {
	// Create the default_team_config_json table, mirroring app_config_json structure
	sql := `
		CREATE TABLE IF NOT EXISTS default_team_config_json (
			id int(10) unsigned NOT NULL UNIQUE default 1,
			json_value JSON NOT NULL,
			created_at timestamp(6) DEFAULT CURRENT_TIMESTAMP(6),
			updated_at timestamp(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (id),
			CONSTRAINT default_team_config_id CHECK (id = 1)
		)
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "create default_team_config_json")
	}

	// Initialize with empty TeamConfig
	// We need to create a minimal valid TeamConfig JSON structure
	defaultConfig := map[string]interface{}{
		"webhook_settings": map[string]interface{}{
			"failing_policies_webhook": map[string]interface{}{
				"enable_failing_policies_webhook": false,
				"destination_url":                 "",
				"policy_ids":                      []int{},
				"host_batch_size":                 0,
			},
			"host_status_webhook": nil,
		},
		"features": map[string]interface{}{
			"enable_host_users":                    true,
			"enable_software_inventory":            true,
			"additional_queries":                   nil,
			"detail_query_overrides":               nil,
			"enable_host_operating_system_details": true,
			"enable_host_software_via_scan":        true,
		},
		"host_expiry_settings": map[string]interface{}{
			"host_expiry_enabled": false,
			"host_expiry_window":  0,
			"jitter_percent":      0,
		},
		"integrations": map[string]interface{}{
			"jira":            nil,
			"zendesk":         nil,
			"google_calendar": nil,
		},
		"mdm": map[string]interface{}{
			"enable_disk_encryption": false,
			"macos_updates": map[string]interface{}{
				"minimum_version": nil,
				"deadline":        nil,
			},
			"windows_updates": map[string]interface{}{
				"deadline_days":     nil,
				"grace_period_days": nil,
			},
			"macos_settings": map[string]interface{}{
				"custom_settings":                nil,
				"enable_end_user_authentication": false,
			},
			"windows_settings": map[string]interface{}{
				"custom_settings": nil,
			},
			"macos_setup": map[string]interface{}{
				"enable_end_user_authentication": false,
				"macos_setup_assistant":          nil,
				"bootstrap_package":              nil,
				"enable_release_device_manually": false,
			},
		},
		"agent_options": nil,
		"scripts":       nil,
		"software":      nil,
	}

	configBytes, err := json.Marshal(defaultConfig)
	if err != nil {
		return errors.Wrap(err, "marshaling default config")
	}

	// Insert the default configuration with fixed timestamps
	_, err = tx.Exec(
		`INSERT INTO default_team_config_json(id, json_value, created_at, updated_at) VALUES(1, ?, '2020-01-01 01:01:01', '2020-01-01 01:01:01')`,
		configBytes,
	)
	if err != nil {
		return errors.Wrap(err, "inserting default team config")
	}

	return nil
}

func Down_20250828120836(_ *sql.Tx) error {
	return nil
}
