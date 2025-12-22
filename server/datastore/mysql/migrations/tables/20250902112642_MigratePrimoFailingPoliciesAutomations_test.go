package tables

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20250902112642(t *testing.T) {
	// Helper function to create test policies
	createTestPolicies := func(t *testing.T, db *sqlx.DB, createGlobal, createNoTeam bool) {
		if createGlobal {
			// Global policies (team_id = NULL)
			_, err := db.Exec(`INSERT INTO policies (id, team_id, name, query, description, checksum) VALUES 
				(1, NULL, 'Global Policy 1', 'SELECT 1', 'Test global policy 1', UNHEX('11111111111111111111111111111111')),
				(2, NULL, 'Global Policy 2', 'SELECT 2', 'Test global policy 2', UNHEX('22222222222222222222222222222222'))`)
			require.NoError(t, err)
		}

		if createNoTeam {
			// No team policies (team_id = 0)
			_, err := db.Exec(`INSERT INTO policies (id, team_id, name, query, description, checksum) VALUES 
				(101, 0, 'No Team Policy 1', 'SELECT 1', 'Test no team policy 1', UNHEX('44444444444444444444444444444444')),
				(102, 0, 'No Team Policy 2', 'SELECT 2', 'Test no team policy 2', UNHEX('55555555555555555555555555555555'))`)
			require.NoError(t, err)
		}
	}

	t.Run("MigrateWithMixedPolicies", func(t *testing.T) {
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
						"url":                             "https://jira1.example.com",
						"username":                        "user1",
						"api_token":                       "token1",
						"project_key":                     "PROJ1",
						"enable_failing_policies":         true,
						"enable_software_vulnerabilities": false,
					},
					map[string]any{
						"url":                             "https://jira2.example.com",
						"username":                        "user2",
						"api_token":                       "token2",
						"project_key":                     "PROJ2",
						"enable_failing_policies":         false,
						"enable_software_vulnerabilities": true,
					},
				},
				"zendesk": []any{
					map[string]any{
						"url":                             "https://zendesk1.example.com",
						"email":                           "email1@example.com",
						"api_token":                       "ztoken1",
						"group_id":                        float64(12345),
						"enable_failing_policies":         true,
						"enable_software_vulnerabilities": false,
					},
					map[string]any{
						"url":                             "https://zendesk2.example.com",
						"email":                           "email2@example.com",
						"api_token":                       "ztoken2",
						"group_id":                        float64(67890),
						"enable_failing_policies":         false,
						"enable_software_vulnerabilities": true,
					},
				},
			},
		}
		appConfigJSON, err := json.Marshal(appConfig)
		require.NoError(t, err)

		_, err = db.Exec(`UPDATE app_config_json SET json_value = ? WHERE id = 1`, appConfigJSON)
		require.NoError(t, err)

		// Create both global and No team policies
		createTestPolicies(t, db, true, true)

		// Test with Primo mode enabled
		t.Setenv("FLEET_PARTNERSHIPS_ENABLE_PRIMO", "true")

		applyNext(t, db)

		// Verify app config has No team policies removed
		var resultJSON json.RawMessage
		err = db.QueryRow(`SELECT json_value FROM app_config_json WHERE id = 1`).Scan(&resultJSON)
		require.NoError(t, err)

		var appResult map[string]any
		err = json.Unmarshal(resultJSON, &appResult)
		require.NoError(t, err)

		// Check that only global policy IDs remain in app config
		webhookSettings, ok := appResult["webhook_settings"].(map[string]any)
		require.True(t, ok)

		failingPoliciesWebhook, ok := webhookSettings["failing_policies_webhook"].(map[string]any)
		require.True(t, ok)

		policyIDs, ok := failingPoliciesWebhook["policy_ids"].([]any)
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

		// Check integrations were copied with only required fields
		integrations, ok := defaultResult["integrations"].(map[string]any)
		require.True(t, ok, "integrations should be present")

		// Check Jira - should have both copied, but only with required fields
		jira, ok := integrations["jira"].([]any)
		require.True(t, ok, "jira should be present")
		assert.Equal(t, 2, len(jira), "should have 2 jira integrations")

		// Check first Jira integration
		jira1, ok := jira[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://jira1.example.com", jira1["url"])
		assert.Equal(t, "PROJ1", jira1["project_key"])
		assert.Equal(t, true, jira1["enable_failing_policies"])
		// These fields should NOT be copied
		assert.Nil(t, jira1["username"], "username should not be copied")
		assert.Nil(t, jira1["api_token"], "api_token should not be copied")
		assert.Nil(t, jira1["enable_software_vulnerabilities"], "enable_software_vulnerabilities should not be copied")

		// Check second Jira integration
		jira2, ok := jira[1].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://jira2.example.com", jira2["url"])
		assert.Equal(t, "PROJ2", jira2["project_key"])
		assert.Equal(t, false, jira2["enable_failing_policies"])

		// Check Zendesk - should have both copied, but only with required fields
		zendesk, ok := integrations["zendesk"].([]any)
		require.True(t, ok, "zendesk should be present")
		assert.Equal(t, 2, len(zendesk), "should have 2 zendesk integrations")

		// Check first Zendesk integration
		zendesk1, ok := zendesk[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://zendesk1.example.com", zendesk1["url"])
		// group_id is saved as int64 but JSON unmarshaling returns float64
		assert.Equal(t, float64(12345), zendesk1["group_id"])
		assert.Equal(t, true, zendesk1["enable_failing_policies"])
		// These fields should NOT be copied
		assert.Nil(t, zendesk1["email"], "email should not be copied")
		assert.Nil(t, zendesk1["api_token"], "api_token should not be copied")
		assert.Nil(t, zendesk1["enable_software_vulnerabilities"], "enable_software_vulnerabilities should not be copied")

		// Check second Zendesk integration
		zendesk2, ok := zendesk[1].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://zendesk2.example.com", zendesk2["url"])
		// group_id is saved as int64 but JSON unmarshaling returns float64
		assert.Equal(t, float64(67890), zendesk2["group_id"])
		assert.Equal(t, false, zendesk2["enable_failing_policies"])
	})

	t.Run("SkipWhenPrimoModeDisabled", func(t *testing.T) {
		db := applyUpToPrev(t)

		// Insert app config with webhook and integrations
		appConfig := map[string]any{
			"webhook_settings": map[string]any{
				"failing_policies_webhook": map[string]any{
					"enable_failing_policies_webhook": true,
					"destination_url":                 "https://example.com/webhook",
					"policy_ids":                      []any{float64(1), float64(101)},
				},
			},
		}
		appConfigJSON, err := json.Marshal(appConfig)
		require.NoError(t, err)

		_, err = db.Exec(`UPDATE app_config_json SET json_value = ? WHERE id = 1`, appConfigJSON)
		require.NoError(t, err)

		// Create both global and No team policies
		createTestPolicies(t, db, true, true)

		var resultJSON json.RawMessage
		err = db.QueryRow(`SELECT json_value FROM default_team_config_json WHERE id = 1`).Scan(&resultJSON)
		require.NoError(t, err)
		var defaultResult map[string]any
		err = json.Unmarshal(resultJSON, &defaultResult)
		require.NoError(t, err)

		// Test with Primo mode disabled (should skip migration)
		_ = os.Unsetenv("FLEET_PARTNERSHIPS_ENABLE_PRIMO")
		applyNext(t, db)

		// Verify configs weren't changed
		err = db.QueryRow(`SELECT json_value FROM app_config_json WHERE id = 1`).Scan(&resultJSON)
		require.NoError(t, err)

		var appResult map[string]any
		err = json.Unmarshal(resultJSON, &appResult)
		require.NoError(t, err)

		// Should still have all original policy IDs in global config
		webhookSettings, _ := appResult["webhook_settings"].(map[string]any)
		failingPoliciesWebhook, _ := webhookSettings["failing_policies_webhook"].(map[string]any)
		policyIDs, _ := failingPoliciesWebhook["policy_ids"].([]any)
		assert.Equal(t, 2, len(policyIDs), "should still have all 2 policy IDs when Primo mode is disabled")

		// Verify default config wasn't changed - it should remain the same
		err = db.QueryRow(`SELECT json_value FROM default_team_config_json WHERE id = 1`).Scan(&resultJSON)
		require.NoError(t, err)
		var defaultResult2 map[string]any
		err = json.Unmarshal(resultJSON, &defaultResult2)
		require.NoError(t, err)
		require.Equal(t, defaultResult, defaultResult2)
	})

	t.Run("OnlyNoTeamPolicies", func(t *testing.T) {
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
						"url":                             "https://jira.example.com",
						"username":                        "user",
						"api_token":                       "token",
						"project_key":                     "TEST",
						"enable_failing_policies":         false,
						"enable_software_vulnerabilities": true,
					},
				},
				"zendesk": []any{
					map[string]any{
						"url":                             "https://zendesk.example.com",
						"email":                           "email@example.com",
						"api_token":                       "token",
						"group_id":                        float64(12345),
						"enable_failing_policies":         true,
						"enable_software_vulnerabilities": false,
					},
				},
			},
		}
		appConfigJSON, err := json.Marshal(appConfig)
		require.NoError(t, err)

		_, err = db.Exec(`UPDATE app_config_json SET json_value = ? WHERE id = 1`, appConfigJSON)
		require.NoError(t, err)

		// Create only No team policies for this test
		createTestPolicies(t, db, false, true)

		// Enable Primo mode and run migration
		t.Setenv("FLEET_PARTNERSHIPS_ENABLE_PRIMO", "true")

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
		// policyIDs should be an empty array
		require.NotNil(t, policyIDs)
		assert.Equal(t, 0, len(policyIDs), "should have no policy IDs remaining in global config")

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

		// Check integrations - both should be copied with only required fields
		integrations, ok := defaultResult["integrations"].(map[string]any)
		require.True(t, ok, "integrations should be present")

		jira, ok := integrations["jira"].([]any)
		require.True(t, ok, "jira should be present")
		assert.Equal(t, 1, len(jira))

		zendesk, ok := integrations["zendesk"].([]any)
		require.True(t, ok, "zendesk should be present")
		assert.Equal(t, 1, len(zendesk))
	})

	t.Run("NoWebhookButHasIntegrations", func(t *testing.T) {
		db := applyUpToPrev(t)

		// Insert app config with no webhook but has integrations
		appConfig := map[string]any{
			"integrations": map[string]any{
				"jira": []any{
					map[string]any{
						"url":                             "https://jira.example.com",
						"username":                        "user",
						"api_token":                       "token",
						"project_key":                     "PROJ",
						"enable_failing_policies":         true,
						"enable_software_vulnerabilities": false,
					},
				},
			},
		}
		appConfigJSON, err := json.Marshal(appConfig)
		require.NoError(t, err)

		_, err = db.Exec(`UPDATE app_config_json SET json_value = ? WHERE id = 1`, appConfigJSON)
		require.NoError(t, err)

		// Enable Primo mode and run migration
		t.Setenv("FLEET_PARTNERSHIPS_ENABLE_PRIMO", "true")

		applyNext(t, db)

		// Verify default config has integrations copied
		var resultJSON json.RawMessage
		err = db.QueryRow(`SELECT json_value FROM default_team_config_json WHERE id = 1`).Scan(&resultJSON)
		require.NoError(t, err)

		var defaultResult map[string]any
		err = json.Unmarshal(resultJSON, &defaultResult)
		require.NoError(t, err)

		// The default config will have webhook_settings from the AddDefaultTeamConfigJSON migration
		// When there's no webhook in global config, our migration shouldn't modify it
		webhookSettings, ok := defaultResult["webhook_settings"].(map[string]any)
		require.True(t, ok, "webhook_settings should exist from the AddDefaultTeamConfigJSON migration")

		failingPoliciesWebhook, ok := webhookSettings["failing_policies_webhook"].(map[string]any)
		require.True(t, ok, "failing_policies_webhook should exist in default state")

		// The failing_policies_webhook should remain in its default state (empty/disabled)
		assert.False(t, failingPoliciesWebhook["enable_failing_policies_webhook"].(bool),
			"failing_policies_webhook should remain disabled when global config has no webhook")
		assert.Empty(t, failingPoliciesWebhook["destination_url"].(string),
			"destination_url should remain empty when global config has no webhook")

		// But integrations should still be copied
		integrations, ok := defaultResult["integrations"].(map[string]any)
		require.True(t, ok, "integrations should be present")

		jira, ok := integrations["jira"].([]any)
		require.True(t, ok, "jira should be present")
		assert.Equal(t, 1, len(jira))

		require.Nil(t, integrations["zendesk"], "zendesk should not be present")

	})
}
