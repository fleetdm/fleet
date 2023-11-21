package tables

import (
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20231004144339(t *testing.T) {
	db := applyUpToPrev(t)

	dataStmts := `
		INSERT INTO teams VALUES
			(1,'2023-07-21 20:32:42','Team 1','','{\"mdm\": {\"macos_setup\": {\"bootstrap_package\": null, \"macos_setup_assistant\": null, \"enable_end_user_authentication\": false}, \"macos_updates\": {\"deadline\": null, \"minimum_version\": null}, \"macos_settings\": {\"custom_settings\": null, \"enable_disk_encryption\": false}}, \"features\": {\"enable_host_users\": true, \"enable_software_inventory\": true}, \"integrations\": {\"jira\": null, \"zendesk\": null}, \"agent_options\": {\"config\": {\"options\": {\"pack_delimiter\": \"/\", \"logger_tls_period\": 10, \"distributed_plugin\": \"tls\", \"disable_distributed\": false, \"logger_tls_endpoint\": \"/api/osquery/log\", \"distributed_interval\": 10, \"distributed_tls_max_attempts\": 3}, \"decorators\": {\"load\": [\"SELECT uuid AS host_uuid FROM system_info;\", \"SELECT hostname AS hostname FROM system_info;\"]}}, \"overrides\": {}}, \"webhook_settings\": {\"failing_policies_webhook\": {\"policy_ids\": null, \"destination_url\": \"\", \"host_batch_size\": 0, \"enable_failing_policies_webhook\": false}}}'),
			(2,'2023-07-21 20:32:47','Team 2','','{\"mdm\": {\"macos_setup\": {\"bootstrap_package\": null, \"macos_setup_assistant\": null, \"enable_end_user_authentication\": false}, \"macos_updates\": {\"deadline\": null, \"minimum_version\": null}, \"macos_settings\": {\"custom_settings\": null, \"enable_disk_encryption\": true}}, \"features\": {\"enable_host_users\": true, \"enable_software_inventory\": true}, \"integrations\": {\"jira\": null, \"zendesk\": null}, \"agent_options\": {\"config\": {\"options\": {\"pack_delimiter\": \"/\", \"logger_tls_period\": 10, \"distributed_plugin\": \"tls\", \"disable_distributed\": false, \"logger_tls_endpoint\": \"/api/osquery/log\", \"distributed_interval\": 10, \"distributed_tls_max_attempts\": 3}, \"decorators\": {\"load\": [\"SELECT uuid AS host_uuid FROM system_info;\", \"SELECT hostname AS hostname FROM system_info;\"]}}, \"overrides\": {}}, \"webhook_settings\": {\"failing_policies_webhook\": {\"policy_ids\": null, \"destination_url\": \"\", \"host_batch_size\": 0, \"enable_failing_policies_webhook\": false}}}');
	`
	_, err := db.Exec(dataStmts)
	require.NoError(t, err)

	var rawConfigs []json.RawMessage
	err = sqlx.Select(db, &rawConfigs, "SELECT config FROM teams ORDER BY id")
	require.NoError(t, err)

	var wantConfigs []map[string]any
	for _, c := range rawConfigs {
		var wantConfig map[string]any
		err = json.Unmarshal(c, &wantConfig)
		require.NoError(t, err)
		wantConfigs = append(wantConfigs, wantConfig)
	}

	applyNext(t, db)

	rawConfigs = []json.RawMessage{}
	err = sqlx.Select(db, &rawConfigs, "SELECT JSON_EXTRACT(config, '$') FROM teams ORDER BY id")
	require.NoError(t, err)

	var gotConfigs []map[string]any
	for _, c := range rawConfigs {
		var gotConfig map[string]any
		err = json.Unmarshal(c, &gotConfig)
		require.NoError(t, err)
		gotConfigs = append(gotConfigs, gotConfig)
	}

	// simulate the ideal behavior with the oldConfigs
	for i, config := range wantConfigs {
		if mdmMap, ok := config["mdm"].(map[string]interface{}); ok {
			// Delete 'mdm.macos_settings.enable_disk_encryption'
			if macosSettings, ok := mdmMap["macos_settings"].(map[string]interface{}); ok {
				delete(macosSettings, "enable_disk_encryption")
			}

			// Set 'mdm.enable_disk_encryption'
			if i == 0 {
				mdmMap["enable_disk_encryption"] = false
			} else {
				mdmMap["enable_disk_encryption"] = true
			}
		}
		wantConfigs[i] = config
	}

	require.ElementsMatch(t, wantConfigs, gotConfigs)
}
