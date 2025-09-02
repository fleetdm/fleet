package tables

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20250902112642(t *testing.T) {
	db := applyUpToPrev(t)

	// Setup: Create test data

	// Insert app config with webhook and integrations
	// Policy IDs 101 and 102 are No team policies, 1 and 2 are global policies
	appConfig := map[string]any{
		"webhook_settings": map[string]any{
			"failing_policies_webhook": map[string]any{
				"enable_failing_policies_webhook": true,
				"destination_url":                 "https://example.com/webhook",
				"host_batch_size":                 100,
				"policy_ids":                      []any{float64(1), float64(2), float64(101), float64(102)},
			},
		},
		"integrations": map[string]any{
			"jira": []any{
				map[string]any{
					"url":                     "https://jira.example.com",
					"project_key":             "TEST",
					"enable_failing_policies": true,
				},
			},
			"zendesk": []any{
				map[string]any{
					"url":                     "https://zendesk.example.com",
					"group_id":                12345,
					"enable_failing_policies": true,
				},
			},
		},
	}
	appConfigJSON, err := json.Marshal(appConfig)
	require.NoError(t, err)

	_, err = db.Exec(`UPDATE app_config_json SET json_value = ? WHERE id = 1`, appConfigJSON)
	require.NoError(t, err)

	// Insert default team config (initially minimal)
	defaultConfig := map[string]any{
		"features": map[string]any{
			"enable_host_users": true,
		},
	}
	defaultConfigJSON, err := json.Marshal(defaultConfig)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO default_team_config_json(id, json_value) VALUES(1, ?) 
		ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`, defaultConfigJSON)
	require.NoError(t, err)

	// Create some policies - both global and No team
	// Global policies
	_, err = db.Exec(`INSERT INTO policies (id, team_id, name, query, description, checksum) VALUES 
		(1, NULL, 'Global Policy 1', 'SELECT 1', 'Test global policy 1', UNHEX('11111111111111111111111111111111')),
		(2, NULL, 'Global Policy 2', 'SELECT 2', 'Test global policy 2', UNHEX('22222222222222222222222222222222'))`)
	require.NoError(t, err)

	// No team policies
	_, err = db.Exec(`INSERT INTO policies (id, team_id, name, query, description, checksum) VALUES 
		(101, 0, 'No Team Policy 1', 'SELECT 1', 'Test no team policy 1', UNHEX('44444444444444444444444444444444')),
		(102, 0, 'No Team Policy 2', 'SELECT 2', 'Test no team policy 2', UNHEX('55555555555555555555555555555555'))`)
	require.NoError(t, err)

	// Test with Primo mode disabled (should skip migration)
	os.Unsetenv("FLEET_PARTNERSHIPS_ENABLE_PRIMO")
	applyNext(t, db)

	// Verify configs weren't changed
	var resultJSON json.RawMessage
	err = db.QueryRow(`SELECT json_value FROM app_config_json WHERE id = 1`).Scan(&resultJSON)
	require.NoError(t, err)

	var appResult map[string]any
	err = json.Unmarshal(resultJSON, &appResult)
	require.NoError(t, err)

	// Should still have all original policy IDs in global config
	webhookSettings, _ := appResult["webhook_settings"].(map[string]any)
	failingPoliciesWebhook, _ := webhookSettings["failing_policies_webhook"].(map[string]any)
	policyIDs, _ := failingPoliciesWebhook["policy_ids"].([]any)
	assert.Equal(t, 4, len(policyIDs), "should still have all 4 policy IDs when Primo mode is disabled")

	// Reset database for next test
	db = applyUpToPrev(t)

	// Re-insert test data
	_, err = db.Exec(`UPDATE app_config_json SET json_value = ? WHERE id = 1`, appConfigJSON)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO default_team_config_json(id, json_value) VALUES(1, ?) 
		ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`, defaultConfigJSON)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO policies (id, team_id, name, query, description, checksum) VALUES 
		(1, NULL, 'Global Policy 1', 'SELECT 1', 'Test global policy 1', UNHEX('11111111111111111111111111111111')),
		(2, NULL, 'Global Policy 2', 'SELECT 2', 'Test global policy 2', UNHEX('22222222222222222222222222222222')),
		(101, 0, 'No Team Policy 1', 'SELECT 1', 'Test no team policy 1', UNHEX('44444444444444444444444444444444')),
		(102, 0, 'No Team Policy 2', 'SELECT 2', 'Test no team policy 2', UNHEX('55555555555555555555555555555555'))`)
	require.NoError(t, err)

	// Test with Primo mode enabled
	os.Setenv("FLEET_PARTNERSHIPS_ENABLE_PRIMO", "true")
	defer os.Unsetenv("FLEET_PARTNERSHIPS_ENABLE_PRIMO")

	applyNext(t, db)

	// Verify app config has No team policies removed
	err = db.QueryRow(`SELECT json_value FROM app_config_json WHERE id = 1`).Scan(&resultJSON)
	require.NoError(t, err)

	err = json.Unmarshal(resultJSON, &appResult)
	require.NoError(t, err)

	// Check that only global policy IDs remain in app config
	webhookSettings, ok := appResult["webhook_settings"].(map[string]any)
	require.True(t, ok)

	failingPoliciesWebhook, ok = webhookSettings["failing_policies_webhook"].(map[string]any)
	require.True(t, ok)

	policyIDs, ok = failingPoliciesWebhook["policy_ids"].([]any)
	require.True(t, ok)
	assert.Equal(t, 2, len(policyIDs), "should have 2 global policy IDs remaining")
	assert.Contains(t, policyIDs, float64(1), "should contain global policy ID 1")
	assert.Contains(t, policyIDs, float64(2), "should contain global policy ID 2")

	// Verify default config has No team policies added
	err = db.QueryRow(`SELECT json_value FROM default_team_config_json WHERE id = 1`).Scan(&resultJSON)
	require.NoError(t, err)

	var defaultResult map[string]any
	err = json.Unmarshal(resultJSON, &defaultResult)
	require.NoError(t, err)

	// Check webhook settings were copied with No team policy IDs
	defaultWebhookSettings, ok := defaultResult["webhook_settings"].(map[string]any)
	require.True(t, ok, "webhook_settings should be present in default config")

	defaultFailingPoliciesWebhook, ok := defaultWebhookSettings["failing_policies_webhook"].(map[string]any)
	require.True(t, ok, "failing_policies_webhook should be present")

	assert.Equal(t, true, defaultFailingPoliciesWebhook["enable_failing_policies_webhook"])
	assert.Equal(t, "https://example.com/webhook", defaultFailingPoliciesWebhook["destination_url"])
	assert.Equal(t, float64(100), defaultFailingPoliciesWebhook["host_batch_size"])

	// Check No team policy IDs
	noTeamPolicyIDs, ok := defaultFailingPoliciesWebhook["policy_ids"].([]any)
	require.True(t, ok, "policy_ids should be present")
	assert.Equal(t, 2, len(noTeamPolicyIDs), "should have 2 No team policy IDs")
	assert.Contains(t, noTeamPolicyIDs, float64(101), "should contain No team policy ID 101")
	assert.Contains(t, noTeamPolicyIDs, float64(102), "should contain No team policy ID 102")

	// Check integrations were copied
	integrations, ok := defaultResult["integrations"].(map[string]any)
	require.True(t, ok, "integrations should be present")

	jira, ok := integrations["jira"].([]any)
	require.True(t, ok, "jira should be present")
	assert.Equal(t, 1, len(jira))

	zendesk, ok := integrations["zendesk"].([]any)
	require.True(t, ok, "zendesk should be present")
	assert.Equal(t, 1, len(zendesk))

	// Verify original features are preserved
	features, ok := defaultResult["features"].(map[string]any)
	require.True(t, ok, "features should still be present")
	assert.Equal(t, true, features["enable_host_users"])
}

func TestUp_20250902112642_OnlyNoTeamPolicies(t *testing.T) {
	db := applyUpToPrev(t)

	// Setup: Create test data with only No team policies in the webhook

	// Insert app config with only No team policy IDs
	appConfig := map[string]any{
		"webhook_settings": map[string]any{
			"failing_policies_webhook": map[string]any{
				"enable_failing_policies_webhook": true,
				"destination_url":                 "https://global.example.com/webhook",
				"policy_ids":                      []any{float64(101), float64(102)},
			},
		},
		"integrations": map[string]any{
			"jira": []any{
				map[string]any{
					"url":                     "https://jira.example.com",
					"project_key":             "TEST",
					"enable_failing_policies": false, // Not enabled, should not be copied
				},
			},
			"zendesk": []any{
				map[string]any{
					"url":                     "https://zendesk.example.com",
					"group_id":                12345,
					"enable_failing_policies": true, // Enabled, should be copied
				},
			},
		},
	}
	appConfigJSON, err := json.Marshal(appConfig)
	require.NoError(t, err)

	_, err = db.Exec(`UPDATE app_config_json SET json_value = ? WHERE id = 1`, appConfigJSON)
	require.NoError(t, err)

	// Insert default team config
	defaultConfig := map[string]any{
		"features": map[string]any{
			"enable_host_users": true,
		},
	}
	defaultConfigJSON, err := json.Marshal(defaultConfig)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO default_team_config_json(id, json_value) VALUES(1, ?) 
		ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`, defaultConfigJSON)
	require.NoError(t, err)

	// Create No team policies
	_, err = db.Exec(`INSERT INTO policies (id, team_id, name, query, description, checksum) VALUES 
		(101, 0, 'No Team Policy 1', 'SELECT 1', 'Test no team policy 1', UNHEX('44444444444444444444444444444444')),
		(102, 0, 'No Team Policy 2', 'SELECT 2', 'Test no team policy 2', UNHEX('55555555555555555555555555555555'))`)
	require.NoError(t, err)

	// Enable Primo mode and run migration
	os.Setenv("FLEET_PARTNERSHIPS_ENABLE_PRIMO", "true")
	defer os.Unsetenv("FLEET_PARTNERSHIPS_ENABLE_PRIMO")

	applyNext(t, db)

	// Verify app config has empty policy IDs
	var resultJSON json.RawMessage
	err = db.QueryRow(`SELECT json_value FROM app_config_json WHERE id = 1`).Scan(&resultJSON)
	require.NoError(t, err)

	var appResult map[string]any
	err = json.Unmarshal(resultJSON, &appResult)
	require.NoError(t, err)

	// Check that policy IDs array is empty in app config
	webhookSettings, ok := appResult["webhook_settings"].(map[string]any)
	require.True(t, ok)

	failingPoliciesWebhook, ok := webhookSettings["failing_policies_webhook"].(map[string]any)
	require.True(t, ok)

	policyIDs, _ := failingPoliciesWebhook["policy_ids"].([]any)
	// policyIDs could be nil or empty array, both are valid
	if policyIDs != nil {
		assert.Equal(t, 0, len(policyIDs), "should have no policy IDs remaining in global config")
	}

	// Verify default config has all the No team policies
	err = db.QueryRow(`SELECT json_value FROM default_team_config_json WHERE id = 1`).Scan(&resultJSON)
	require.NoError(t, err)

	var defaultResult map[string]any
	err = json.Unmarshal(resultJSON, &defaultResult)
	require.NoError(t, err)

	// Check webhook settings
	defaultWebhookSettings, ok := defaultResult["webhook_settings"].(map[string]any)
	require.True(t, ok)

	defaultFailingPoliciesWebhook, ok := defaultWebhookSettings["failing_policies_webhook"].(map[string]any)
	require.True(t, ok)

	// Should have the global config URL
	assert.Equal(t, "https://global.example.com/webhook", defaultFailingPoliciesWebhook["destination_url"])
	noTeamPolicyIDs, ok := defaultFailingPoliciesWebhook["policy_ids"].([]any)
	require.True(t, ok)
	assert.Equal(t, 2, len(noTeamPolicyIDs))
	assert.Contains(t, noTeamPolicyIDs, float64(101))
	assert.Contains(t, noTeamPolicyIDs, float64(102))

	// Check integrations - both Jira and Zendesk should be copied regardless of enable_failing_policies
	integrations, ok := defaultResult["integrations"].(map[string]any)
	require.True(t, ok, "integrations should be present")

	jira, ok := integrations["jira"].([]any)
	require.True(t, ok, "jira should be present")
	assert.Equal(t, 1, len(jira))

	zendesk, ok := integrations["zendesk"].([]any)
	require.True(t, ok, "zendesk should be present")
	assert.Equal(t, 1, len(zendesk))
}
