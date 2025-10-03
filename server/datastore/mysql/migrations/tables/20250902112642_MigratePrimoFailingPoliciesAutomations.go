package tables

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20250902112642, Down_20250902112642)
}

func Up_20250902112642(tx *sql.Tx) error {
	// Only run this migration if FLEET_PARTNERSHIPS_ENABLE_PRIMO is set to true
	enablePrimo := os.Getenv("FLEET_PARTNERSHIPS_ENABLE_PRIMO")
	if enablePrimo != "true" && enablePrimo != "1" {
		// Skip migration if not in Primo mode
		return nil
	}

	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// Step 1: Read global config from app_config_json
	var appConfigJSON json.RawMessage
	err := txx.Get(&appConfigJSON, "SELECT json_value FROM app_config_json LIMIT 1")
	if err != nil {
		return fmt.Errorf("selecting app config: %w", err)
	}

	var appConfig map[string]any
	if err := json.Unmarshal(appConfigJSON, &appConfig); err != nil {
		return fmt.Errorf("unmarshal app config: %w", err)
	}

	// Extract webhook settings and integrations from global config
	webhookSettings, _ := appConfig["webhook_settings"].(map[string]any)
	integrations, _ := appConfig["integrations"].(map[string]any)

	var globalFailingPoliciesWebhook map[string]any
	if webhookSettings != nil {
		globalFailingPoliciesWebhook, _ = webhookSettings["failing_policies_webhook"].(map[string]any)
	}

	noTeamIDs := []any{} // Initialize as empty array
	var needToUpdateAppConfig bool

	// Process webhook if configured
	if globalFailingPoliciesWebhook != nil {
		// Step 2: Get all No team policy IDs
		var noTeamPolicyIDs []uint
		err = txx.Select(&noTeamPolicyIDs, "SELECT id FROM policies WHERE team_id = 0")
		if err != nil {
			return fmt.Errorf("selecting no team policy IDs: %w", err)
		}

		// Create a set for quick lookup
		noTeamPolicySet := make(map[uint]struct{})
		for _, id := range noTeamPolicyIDs {
			noTeamPolicySet[id] = struct{}{}
		}

		// Step 3: Separate policy IDs into global and No team
		globalPolicyIDs, _ := globalFailingPoliciesWebhook["policy_ids"].([]any)
		remainingGlobalIDs := []any{} // Initialize as empty array to preserve JSON array structure

		for _, policyIDInterface := range globalPolicyIDs {
			// Handle different number types that might come from JSON
			var policyID uint
			switch v := policyIDInterface.(type) {
			case float64:
				policyID = uint(v)
			case int:
				if v < 0 {
					continue
				}
				policyID = uint(v)
			case uint:
				policyID = v
			default:
				continue
			}

			if _, ok := noTeamPolicySet[policyID]; ok {
				// This is a No team policy
				noTeamIDs = append(noTeamIDs, policyID)
			} else {
				// This is a non-No team policy
				remainingGlobalIDs = append(remainingGlobalIDs, policyIDInterface)
			}
		}

		// Step 4: Update global config to remove No team policy IDs
		globalFailingPoliciesWebhook["policy_ids"] = remainingGlobalIDs
		webhookSettings["failing_policies_webhook"] = globalFailingPoliciesWebhook
		appConfig["webhook_settings"] = webhookSettings
		needToUpdateAppConfig = true
	}

	// Save updated app config if needed
	if needToUpdateAppConfig {
		updatedAppConfigJSON, err := json.Marshal(appConfig)
		if err != nil {
			return fmt.Errorf("marshal updated app config: %w", err)
		}

		_, err = tx.Exec("UPDATE app_config_json SET json_value = ?", updatedAppConfigJSON)
		if err != nil {
			return fmt.Errorf("updating app config: %w", err)
		}
	}

	// Step 5: Read current default_team_config_json
	var defaultConfigJSON json.RawMessage
	err = txx.Get(&defaultConfigJSON, "SELECT json_value FROM default_team_config_json WHERE id = 1")
	if err != nil {
		return fmt.Errorf("selecting default team config: %w", err)
	}

	var defaultConfig map[string]any
	if err := json.Unmarshal(defaultConfigJSON, &defaultConfig); err != nil {
		return fmt.Errorf("unmarshal default team config: %w", err)
	}

	// Step 6: Update default team config with No team automation settings
	if globalFailingPoliciesWebhook != nil {
		// Set up webhook settings for No team
		defaultWebhookSettings := make(map[string]any)

		// Copy global failing policies webhook settings
		newFailingPoliciesWebhook := make(map[string]any)

		// Copy all settings except policy_ids
		for k, v := range globalFailingPoliciesWebhook {
			if k != "policy_ids" {
				newFailingPoliciesWebhook[k] = v
			}
		}

		// Set the No team policy IDs (could be empty array)
		newFailingPoliciesWebhook["policy_ids"] = noTeamIDs
		defaultWebhookSettings["failing_policies_webhook"] = newFailingPoliciesWebhook
		defaultConfig["webhook_settings"] = defaultWebhookSettings
	}

	// Handle integrations (Jira and Zendesk) - copy only required fields
	if integrations != nil {
		defaultIntegrations := make(map[string]any)

		// Process Jira integrations
		if globalJira, ok := integrations["jira"].([]any); ok && len(globalJira) > 0 {
			var jiraForNoTeam []any
			for _, jiraConfig := range globalJira {
				if config, ok := jiraConfig.(map[string]any); ok {
					// Create a copy for No team with only required fields
					noTeamJira := make(map[string]any)
					if url, ok := config["url"].(string); ok {
						noTeamJira["url"] = url
					}
					if projectKey, ok := config["project_key"].(string); ok {
						noTeamJira["project_key"] = projectKey
					}
					if enableFailingPolicies, ok := config["enable_failing_policies"].(bool); ok {
						noTeamJira["enable_failing_policies"] = enableFailingPolicies
					}
					jiraForNoTeam = append(jiraForNoTeam, noTeamJira)
				}
			}
			if len(jiraForNoTeam) > 0 {
				defaultIntegrations["jira"] = jiraForNoTeam
			}
		}

		// Process Zendesk integrations
		if globalZendesk, ok := integrations["zendesk"].([]any); ok && len(globalZendesk) > 0 {
			var zendeskForNoTeam []any
			for _, zendeskConfig := range globalZendesk {
				if config, ok := zendeskConfig.(map[string]any); ok {
					// Create a copy for No team with only required fields
					noTeamZendesk := make(map[string]any)
					if url, ok := config["url"].(string); ok {
						noTeamZendesk["url"] = url
					}
					// Handle group_id which could be int64 or float64 from JSON
					if groupID, ok := config["group_id"].(float64); ok {
						noTeamZendesk["group_id"] = int64(groupID)
					} else if groupID, ok := config["group_id"].(int64); ok {
						noTeamZendesk["group_id"] = groupID
					}
					if enableFailingPolicies, ok := config["enable_failing_policies"].(bool); ok {
						noTeamZendesk["enable_failing_policies"] = enableFailingPolicies
					}
					zendeskForNoTeam = append(zendeskForNoTeam, noTeamZendesk)
				}
			}
			if len(zendeskForNoTeam) > 0 {
				defaultIntegrations["zendesk"] = zendeskForNoTeam
			}
		}

		if len(defaultIntegrations) > 0 {
			defaultConfig["integrations"] = defaultIntegrations
		}
	}

	// Step 7: Save updated default team config
	updatedDefaultConfigJSON, err := json.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("marshal updated default team config: %w", err)
	}

	_, err = tx.Exec(
		`UPDATE default_team_config_json SET json_value = ? WHERE id = 1`,
		updatedDefaultConfigJSON,
	)
	if err != nil {
		return fmt.Errorf("updating default team config: %w", err)
	}

	return nil
}

func Down_20250902112642(_ *sql.Tx) error {
	// No down migration needed
	return nil
}
