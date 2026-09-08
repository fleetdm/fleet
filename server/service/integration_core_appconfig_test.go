package service

// App configuration tests for the core (no-license) suite.
//
// Belongs here: reading and patching the global config, including default values,
// deprecated-field handling and historical data; the org logo; external
// integrations (Jira/Zendesk) and Google Calendar; and premium gating of config
// fields on an unlicensed deployment.
//
// Does not belong here: the automation/webhook sections of the config
// (integration_core_activities_test.go), and broad license-gating sweeps across
// unrelated endpoints (integration_core_misc_test.go).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestAppConfigAdditionalQueriesCanBeRemoved() {
	t := s.T()

	spec := []byte(`
  host_expiry_settings:
    host_expiry_enabled: true
    host_expiry_window: 0
  features:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
`)
	s.applyConfig(spec)

	spec = []byte(`
  features:
    enable_host_users: true
    additional_queries: null
`)
	s.applyConfig(spec)

	config := s.getConfig()
	assert.Nil(t, config.Features.AdditionalQueries)
	assert.True(t, config.HostExpirySettings.HostExpiryEnabled)
}

func (s *integrationTestSuite) TestAppConfigDetailQueriesOverrides() {
	t := s.T()

	spec := []byte(`
  features:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
    detail_query_overrides:
      users: null
      software_linux: "select * from blah;"
`)
	s.applyConfig(spec)

	config := s.getConfig()
	require.NotNil(t, config.Features.DetailQueryOverrides)
	require.Nil(t, config.Features.DetailQueryOverrides["users"])
	require.NotNil(t, config.Features.DetailQueryOverrides["software_linux"])
	require.Equal(t, "select * from blah;", *config.Features.DetailQueryOverrides["software_linux"])
}

func (s *integrationTestSuite) TestAppConfigDefaultValues() {
	config := s.getConfig()
	s.Run("Update interval", func() {
		s.Require().Equal(1*time.Hour, config.UpdateInterval.OSQueryDetail)
	})

	s.Run("has logging", func() {
		s.Require().NotNil(config.Logging)
	})
}

func (s *integrationTestSuite) TestAppConfigDeprecatedFields() {
	t := s.T()

	spec := []byte(`
  host_settings:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
    enable_software_inventory: true
`)
	s.applyConfig(spec)
	config := s.getConfig()
	require.NotNil(t, config.Features.AdditionalQueries)
	require.True(t, config.Features.EnableHostUsers)
	require.True(t, config.Features.EnableSoftwareInventory)

	spec = []byte(`
  host_settings:
    additional_queries: null
    enable_host_users: false
    enable_software_inventory: false
`)
	s.applyConfig(spec)
	config = s.getConfig()
	require.Nil(t, config.Features.AdditionalQueries)
	require.False(t, config.Features.EnableHostUsers)
	require.False(t, config.Features.EnableSoftwareInventory)

	// Test raw API interactions
	appConfigSpec := map[string]map[string]bool{
		"host_settings":   {"enable_software_inventory": true},
		"server_settings": {"enable_analytics": false},
	}
	s.Do("PATCH", "/api/latest/fleet/config", appConfigSpec, http.StatusOK)
	config = s.getConfig()
	require.True(t, config.Features.EnableSoftwareInventory)

	// Skip our serialization mechanism, to make sure an old config stored in the DB is still valid
	var previousRawConfig string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		err := sqlx.GetContext(context.Background(), q, &previousRawConfig, "SELECT json_value FROM app_config_json")
		if err != nil {
			return err
		}
		insertAppConfigQuery := `INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`
		_, err = q.ExecContext(context.Background(), insertAppConfigQuery, `
    {
      "host_settings": {
        "enable_host_users": false,
        "enable_software_inventory": true,
        "additional_queries": { "foo": "bar" }
      }
    }`)
		return err
	})

	var resp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &resp)
	require.False(t, resp.Features.EnableHostUsers)
	require.True(t, resp.Features.EnableSoftwareInventory)
	require.NotNil(t, resp.Features.AdditionalQueries)

	// restore the previous appconfig so that other tests are not impacted
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		insertAppConfigQuery := `INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`
		_, err := q.ExecContext(context.Background(), insertAppConfigQuery, previousRawConfig)
		return err
	})
}

func (s *integrationTestSuite) TestAppConfigHistoricalData() {
	t := s.T()
	ctx := context.Background()

	// Ensure a known starting state — earlier tests in this suite may have
	// PATCHed the AppConfig (the suite shares state), so an earlier no-op
	// SaveAppConfig with a zero-value Features could have stamped
	// historical_data={false,false} into the stored JSON.
	s.Do("PATCH", "/api/latest/fleet/config",
		map[string]any{"features": map[string]any{"historical_data": map[string]any{"uptime": true, "vulnerabilities": true}}},
		http.StatusOK)
	cfg := s.getConfig()
	require.True(t, cfg.Features.HistoricalData.Uptime)
	require.True(t, cfg.Features.HistoricalData.Vulnerabilities)

	// PATCH only the vulnerabilities sub-key — uptime SHALL remain true.
	// Snapshot the most recent activity ID (any type) as a watermark so we can
	// confirm a new disabled_historical_dataset row is actually emitted.
	preDisableWatermark := s.lastActivityMatches("", "", 0)
	s.Do("PATCH", "/api/latest/fleet/config",
		map[string]any{"features": map[string]any{"historical_data": map[string]any{"vulnerabilities": false}}},
		http.StatusOK)
	cfg = s.getConfig()
	require.True(t, cfg.Features.HistoricalData.Uptime, "uptime preserved when omitted from PATCH")
	require.False(t, cfg.Features.HistoricalData.Vulnerabilities)

	// A new disabled_historical_dataset activity for vulnerabilities, no fleet scope.
	require.Greater(t, s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(),
		`{"dataset":"vulnerabilities","fleet_id":null,"fleet_name":null}`,
		0,
	), preDisableWatermark, "new disable activity emitted for PATCH")

	// PATCH the same value back — no new activity should be emitted.
	priorActivityID := s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(), "", 0,
	)
	s.Do("PATCH", "/api/latest/fleet/config",
		map[string]any{"features": map[string]any{"historical_data": map[string]any{"vulnerabilities": false}}},
		http.StatusOK)
	require.Equal(t, priorActivityID, s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(), "", 0,
	), "no new activity for no-op PATCH")

	// Flip both in one PATCH — re-enable vulnerabilities, disable uptime → 2 activities.
	// Use the most recent activity ID (any type) as a watermark; the new
	// enabled/disabled activities for this PATCH must have IDs greater than it.
	preFlipWatermark := s.lastActivityMatches("", "", 0)
	s.Do("PATCH", "/api/latest/fleet/config",
		map[string]any{"features": map[string]any{"historical_data": map[string]any{"uptime": false, "vulnerabilities": true}}},
		http.StatusOK)
	cfg = s.getConfig()
	require.False(t, cfg.Features.HistoricalData.Uptime)
	require.True(t, cfg.Features.HistoricalData.Vulnerabilities)
	require.Greater(t, s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEnabledHistoricalDataset{}.ActivityName(),
		`{"dataset":"vulnerabilities","fleet_id":null,"fleet_name":null}`,
		0,
	), preFlipWatermark, "new enable activity emitted for vulnerabilities re-enable")
	require.Greater(t, s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(),
		`{"dataset":"uptime","fleet_id":null,"fleet_name":null}`,
		0,
	), preFlipWatermark, "new disable activity emitted for uptime")

	// Existing rows whose stored JSON omits historical_data SHALL read back
	// with both sub-keys true. Simulate a pre-change deployment by writing a
	// row whose features block lacks the key, then verify that AppConfig
	// reads back with defaults applied.
	var previousRawConfig string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		if err := sqlx.GetContext(ctx, q, &previousRawConfig, "SELECT json_value FROM app_config_json"); err != nil {
			return err
		}
		preChangeJSON := []byte(`{"features": {"enable_host_users": true, "enable_software_inventory": false, "additional_queries": null}}`)
		_, err := q.ExecContext(ctx,
			`INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`,
			preChangeJSON)
		return err
	})
	t.Cleanup(func() {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`,
				previousRawConfig)
			return err
		})
	})

	loadedCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	require.True(t, loadedCfg.Features.HistoricalData.Uptime, "pre-change row reads back as default true")
	require.True(t, loadedCfg.Features.HistoricalData.Vulnerabilities, "pre-change row reads back as default true")
}

func (s *integrationTestSuite) TestExternalIntegrationsConfig() {
	t := s.T()

	// create a test http server to act as the Jira and Zendesk server
	srvURL := startExternalServiceWebServer(t)

	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusOK)

	config := s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, srvURL, config.Integrations.Jira[0].URL)
	require.Equal(t, "ok", config.Integrations.Jira[0].Username)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Jira[0].APIToken)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// add a second, disabled Jira integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 2)

	// first integration
	require.Equal(t, srvURL, config.Integrations.Jira[0].URL)
	require.Equal(t, "ok", config.Integrations.Jira[0].Username)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Jira[0].APIToken)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// second integration
	require.Equal(t, srvURL, config.Integrations.Jira[1].URL)
	require.Equal(t, "ok", config.Integrations.Jira[1].Username)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Jira[1].APIToken)
	require.Equal(t, "qux2", config.Integrations.Jira[1].ProjectKey)
	require.False(t, config.Integrations.Jira[1].EnableSoftwareVulnerabilities)

	// make an unrelated appconfig change, should not remove the integrations
	var appCfgResp appConfigResponse
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"org_info": {
			"org_name": "test-integrations"
		}
	}`), http.StatusOK, &appCfgResp)
	require.Equal(t, "test-integrations", appCfgResp.OrgInfo.OrgName)
	require.Len(t, appCfgResp.Integrations.Jira, 2)

	// delete first Jira integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux2", config.Integrations.Jira[0].ProjectKey)

	// replace Jira integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "ok",
					"project_key": "qux",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.False(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// try adding Jira integration without sending API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "ok",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusBadRequest)

	// try adding Jira integration with masked API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "ok",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": %q,
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusBadRequest)

	// edit Jira integration without sending API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// edit Jira integration with masked API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": %q,
					"project_key": "qux",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.False(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// edit Jira integration sending explicit "" as API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// unknown fields fails as bad request
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"UNKNOWN_FIELD": "foo"
			}]
		}
	}`, srvURL)), http.StatusBadRequest)

	// unknown project key fails as bad request
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": %q,
					"project_key": "qux3",
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusBadRequest)

	// cannot have two integrations enabled at the same time
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "bar2",
					"project_key": "qux2",
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// cannot have two jira integrations with the same project key
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "bar2",
					"project_key": "qux",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// even disabled integrations are tested for Jira connection and credentials,
	// so this fails because the 2nd one uses the "fail" username.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "fail",
					"api_token": "bar2",
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusBadRequest)

	// cannot enable webhook with a jira integration already enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`), http.StatusUnprocessableEntity)

	// disable jira, now we can enable webhook
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
		"jira": [{
			"url": %q,
			"username": "ok",
			"api_token": "bar",
			"project_key": "qux",
			"enable_software_vulnerabilities": false
		}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusOK)

	// cannot enable jira with webhook already enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// disable webhook, enable jira with wrong credentials
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "fail",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusBadRequest)

	// update jira config to correct credentials (need to disable webhook too as
	// last request failed)
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusOK)

	// if no jira nor zendesk integrations are provided, does not remove integrations
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"integrations": {}
	}`), http.StatusOK, &appCfgResp)
	require.Len(t, appCfgResp.Integrations.Jira, 1)

	// if explicitly-empty arrays are provided, remove all integrations
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"integrations": {
			"jira": [],
			"zendesk": []
		}
	}`), http.StatusOK, &appCfgResp)
	require.Empty(t, appCfgResp.Integrations.Jira)

	// set environmental varible to use Zendesk test client
	t.Setenv("TEST_ZENDESK_CLIENT", "true")
	// create zendesk integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Empty(t, config.Integrations.Jira)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, srvURL, config.Integrations.Zendesk[0].URL)
	require.Equal(t, "ok@example.com", config.Integrations.Zendesk[0].Email)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Zendesk[0].APIToken)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// add a second, disabled Zendesk integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "test123@example.com",
					"api_token": "ok",
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Empty(t, config.Integrations.Jira)
	require.Len(t, config.Integrations.Zendesk, 2)

	// first integration
	require.Equal(t, srvURL, config.Integrations.Zendesk[0].URL)
	require.Equal(t, "ok@example.com", config.Integrations.Zendesk[0].Email)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Zendesk[0].APIToken)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// second integration
	require.Equal(t, srvURL, config.Integrations.Zendesk[1].URL)
	require.Equal(t, "test123@example.com", config.Integrations.Zendesk[1].Email)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Zendesk[1].APIToken)
	require.Equal(t, int64(123), config.Integrations.Zendesk[1].GroupID)
	require.False(t, config.Integrations.Zendesk[1].EnableSoftwareVulnerabilities)

	// make an unrelated appconfig change, should not remove the integrations
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"org_info": {
			"org_name": "test-integrations-zendesk"
		}
	}`), http.StatusOK, &appCfgResp)
	require.Equal(t, "test-integrations-zendesk", appCfgResp.OrgInfo.OrgName)
	require.Len(t, appCfgResp.Integrations.Zendesk, 2)

	// delete first Zendesk integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "test123@example.com",
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Empty(t, config.Integrations.Jira)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(123), config.Integrations.Zendesk[0].GroupID)

	// replace Zendesk integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Empty(t, config.Integrations.Jira)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.False(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// try adding Zendesk integration without sending API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "test123@example.com",
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusBadRequest)

	// try adding Zendesk integration with masked API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "test123@example.com",
					"api_token": %q,
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusBadRequest)

	// edit Zendesk integration without sending API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Empty(t, config.Integrations.Jira)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// edit Zendesk integration with masked API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": %q,
					"group_id": 122,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusOK)

	config = s.getConfig()
	require.Empty(t, config.Integrations.Jira)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.False(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// edit Zendesk integration with explicit "" API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Empty(t, config.Integrations.Jira)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// unknown fields fails as bad request
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"UNKNOWN_FIELD": "foo"
			}]
		}
	}`, srvURL)), http.StatusBadRequest)

	// unknown group id fails as bad request
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 999,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusBadRequest)

	// cannot have two zendesk integrations enabled at the same time
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "not.ok@example.com",
					"api_token": "ok",
					"group_id": 123,
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// cannot have two zendesk integrations with the same group id
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "not.ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// even disabled integrations are tested for Zendesk connection and credentials,
	// so this fails because the 2nd one uses the "fail" token.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "not.ok@example.com",
					"api_token": "fail",
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusBadRequest)

	// cannot enable webhook with a zendesk integration already enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`), http.StatusUnprocessableEntity)

	// disable zendesk, now we can enable webhook
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": false
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusOK)

	// cannot enable zendesk with webhook already enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// disable webhook, enable zendesk with wrong credentials
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "not.ok@example.com",
				"api_token": "fail",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusBadRequest)

	// update zendesk config to correct credentials (need to disable webhook too as
	// last request failed)
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusOK)

	// can have jira enabled and zendesk disabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}],
			"zendesk": [{
				"url": %[1]q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": false
			}]
		}
	}`, srvURL)), http.StatusOK)
	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.False(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// can have jira disabled and zendesk enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": false
			}],
			"zendesk": [{
				"url": %[1]q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusOK)
	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.False(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// cannot have both jira enabled and zendesk enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}],
			"zendesk": [{
				"url": %[1]q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// if no jira nor zendesk integrations are provided, does not remove integrations
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"integrations": {}
	}`), http.StatusOK, &appCfgResp)
	require.Len(t, appCfgResp.Integrations.Jira, 1)
	require.Len(t, appCfgResp.Integrations.Zendesk, 1)

	// remove all integrations on exit, so that other tests can enable the
	// webhook as needed
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"integrations": {
		"jira": [],
		"zendesk": []
		}
	}`), http.StatusOK)
	config = s.getConfig()
	require.Empty(t, config.Integrations.Jira)
	require.Empty(t, config.Integrations.Zendesk)

	// enable webhooks
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "http://some/url"
    			},
	    		"failing_policies_webhook": {
	     	 		"enable_failing_policies_webhook": true,
     		 		"destination_url": "http://some/url",
				"host_batch_size": 1000
	    		},
	    		"host_status_webhook": {
	     	 		"enable_host_status_webhook": true,
	     	 		"destination_url": "http://some/url",
					  "host_percentage": 2,
						"days_count": 1
	    		}
		}
	}`), http.StatusOK)
	config = s.getConfig()
	require.True(t, config.WebhookSettings.ActivitiesWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.ActivitiesWebhook.DestinationURL)
	require.True(t, config.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	require.True(t, config.WebhookSettings.HostStatusWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.HostStatusWebhook.DestinationURL)
}

func (s *integrationTestSuite) TestGoogleCalendarIntegrations() {
	t := s.T()
	email := "service-account@example.com"
	privateKey := "-----BEGIN PRIVATE KEY-----\nXXXXX\n-----END"
	domain := "example.com"
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": %q
			}]
		}
	}`, email, privateKey, domain,
		)), http.StatusOK,
	)

	appConfig := s.getConfig()
	require.Len(t, appConfig.Integrations.GoogleCalendar, 1)
	assert.True(t, appConfig.Integrations.GoogleCalendar[0].ApiKey.IsMasked())
	assert.Equal(t, domain, appConfig.Integrations.GoogleCalendar[0].Domain)

	// Add 2nd config -- not allowed at this time
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": %q
			},
			{
				"api_key_json": {
					"client_email": "bozo@example.com",
					"private_key": "abc"
				},
				"domain": "example.com"
			}]
		}
	}`, email, privateKey, domain,
		)), http.StatusUnprocessableEntity,
	)

	// Make an unrelated config change, should not remove the integrations
	var appCfgResp appConfigResponse
	s.DoJSON(
		"PATCH", "/api/v1/fleet/config", json.RawMessage(
			`{
		"org_info": {
			"org_name": "test-google-calendar-integrations"
		}
	}`,
		), http.StatusOK, &appCfgResp,
	)
	require.Equal(t, "test-google-calendar-integrations", appCfgResp.OrgInfo.OrgName)
	require.Len(t, appCfgResp.Integrations.GoogleCalendar, 1)

	// Update calendar config
	domain = "new.com"
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": %q
			}]
		}
	}`, email, privateKey, domain,
		)), http.StatusOK,
	)
	appConfig = s.getConfig()
	require.Len(t, appConfig.Integrations.GoogleCalendar, 1)
	assert.True(t, appConfig.Integrations.GoogleCalendar[0].ApiKey.IsMasked())
	assert.Equal(t, domain, appConfig.Integrations.GoogleCalendar[0].Domain)

	// Clearing other integrations does not clear Google Calendar integration
	appCfgResp = appConfigResponse{}
	s.DoJSON(
		"PATCH", "/api/v1/fleet/config", json.RawMessage(
			`{
		"integrations": {
			"jira": [],
			"zendesk": []
		}
	}`,
		), http.StatusOK, &appCfgResp,
	)
	require.Len(t, appCfgResp.Integrations.GoogleCalendar, 1)

	// Clearing Google Calendar integration
	appCfgResp = appConfigResponse{}
	s.DoJSON(
		"PATCH", "/api/v1/fleet/config", json.RawMessage(
			`{
		"integrations": {
			"google_calendar": []
		}
	}`,
		), http.StatusOK, &appCfgResp,
	)
	assert.Empty(t, appCfgResp.Integrations.GoogleCalendar)

	// Try adding Google Calendar integration without sending private key -- not allowed
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q
				},
				"domain": %q
			}]
		}
	}`, email, domain,
		)), http.StatusUnprocessableEntity,
	)

	// Empty email -- not allowed
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": " ",
					"private_key": %q
				},
				"domain": %q
			}]
		}
	}`, privateKey, domain,
		)), http.StatusUnprocessableEntity,
	)

	// Empty domain -- not allowed
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": ""
			}]
		}
	}`, email, privateKey,
		)), http.StatusUnprocessableEntity,
	)

	// Unknown fields fails as bad request
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": %q,
				"foo": "bar"
			}]
		}
	}`, email, privateKey, domain,
		)), http.StatusBadRequest,
	)

	// Null api_key_json -- fails validation
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": null,
				"domain": %q
			}]
		}
	}`, domain,
		)), http.StatusUnprocessableEntity,
	)
}

// Free tier must not expose or accept fleet_desktop.sso_enabled, and must not
// let a value stored under a premium license (before a downgrade) block
// unrelated config changes.
func (s *integrationTestSuite) TestFleetDesktopSSOFreeTier() {
	t := s.T()
	ctx := t.Context()

	// PATCH attempting to enable it is rejected with the license error
	res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"fleet_desktop":{"sso_enabled":true}}`), http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "missing or invalid license")

	// seed a value as if it had been set while premium, then downgraded
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.FleetDesktop.SSOEnabled = true
	require.NoError(t, s.ds.SaveAppConfig(ctx, appCfg))

	// GET /config rebuilds FleetDesktopSettings field by field and premium-gates
	// the values, so the stored flag must read back as false here
	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.False(t, acResp.FleetDesktop.SSOEnabled)

	// an unrelated PATCH still succeeds and resets the stored value rather than
	// failing on a premium-only setting left over from the downgrade
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"host_expiry_settings":{"host_expiry_enabled":false}}`), http.StatusOK, &acResp)
	require.False(t, acResp.FleetDesktop.SSOEnabled)

	appCfg, err = s.ds.AppConfig(ctx)
	require.NoError(t, err)
	require.False(t, appCfg.FleetDesktop.SSOEnabled)
}

func (s *integrationTestSuite) TestAppConfig() {
	t := s.T()
	ctx := context.Background()

	// get the app config
	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Equal(t, "free", acResp.License.Tier)
	assert.Equal(t, "FleetTest", acResp.OrgInfo.OrgName) // set in SetupSuite
	assert.False(t, acResp.MDM.AppleBMTermsExpired)
	assert.False(t, acResp.ActivityExpirySettings.ActivityExpiryEnabled)
	assert.Zero(t, acResp.ActivityExpirySettings.ActivityExpiryWindow)
	assert.False(t, acResp.ServerSettings.AIFeaturesDisabled)
	assert.False(t, acResp.GitOpsConfig.GitopsModeEnabled)
	assert.Empty(t, acResp.GitOpsConfig.RepositoryURL)
	expectedMaxPackageSize := config.TestConfig().Server.MaxInstallerSizeBytes
	assert.Equal(t, expectedMaxPackageSize, acResp.MaxSoftwarePackageSize)

	// set the apple BM terms expired flag, and the enabled and configured flags,
	// we'll check again at the end of this test to make sure they weren't
	// modified by any PATCH request (it cannot be set via this endpoint).
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.MDM.AppleBMTermsExpired = true
	appCfg.MDM.AppleBMEnabledAndConfigured = true
	appCfg.MDM.EnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)

	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.AppleBMTermsExpired)
	assert.True(t, acResp.MDM.AppleBMEnabledAndConfigured)
	assert.True(t, acResp.MDM.EnabledAndConfigured)

	// no server settings set for the URL, so not possible to test the
	// certificate endpoint
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "org_info": {
        "org_name": "test"
    }
  }`), http.StatusOK, &acResp)
	assert.Equal(t, "test", acResp.OrgInfo.OrgName)
	assert.True(t, acResp.MDM.AppleBMTermsExpired)

	// the global agent options were not modified by the last call, so the
	// corresponding activity should not have been created.
	var listActivities listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listActivities, "order_key", "id", "order_direction", "desc")
	if len(listActivities.Activities) > 1 {
		// if there is an activity, make sure it is not edited_agent_options
		require.NotEqual(t, fleet.ActivityTypeEditedAgentOptions{}.ActivityName(), listActivities.Activities[0].Type)
	}

	// and it did not update the appconfig
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"logger_plugin": "tls"`) // default agent options has this setting

	// Invalid activity expiry window.
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "activity_expiry_settings": {
        "activity_expiry_enabled": true,
        "activity_expiry_window": -1
    }
  }`), http.StatusUnprocessableEntity, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.False(t, acResp.ActivityExpirySettings.ActivityExpiryEnabled)
	require.Zero(t, acResp.ActivityExpirySettings.ActivityExpiryWindow)

	// Valid activity expiry window.
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "activity_expiry_settings": {
        "activity_expiry_enabled": true,
        "activity_expiry_window": 42
    }
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.ActivityExpirySettings.ActivityExpiryEnabled)
	require.Equal(t, 42, acResp.ActivityExpirySettings.ActivityExpiryWindow)

	// preserve_host_activities_on_reenrollment round-trip.
	initialPreserve := acResp.ActivityExpirySettings.PreserveHostActivitiesOnReenrollment
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "activity_expiry_settings": {
        "preserve_host_activities_on_reenrollment": true
    }
  }`), http.StatusOK, &acResp)
	require.True(t, acResp.ActivityExpirySettings.PreserveHostActivitiesOnReenrollment)
	require.True(t, acResp.ActivityExpirySettings.ActivityExpiryEnabled)
	require.Equal(t, 42, acResp.ActivityExpirySettings.ActivityExpiryWindow)

	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "activity_expiry_settings": {
        "preserve_host_activities_on_reenrollment": false
    }
  }`), http.StatusOK, &acResp)
	require.False(t, acResp.ActivityExpirySettings.PreserveHostActivitiesOnReenrollment)

	// Restore initial value to keep subsequent tests order-independent.
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
    "activity_expiry_settings": {
        "preserve_host_activities_on_reenrollment": %t
    }
  }`, initialPreserve)), http.StatusOK, &acResp)
	require.Equal(t, initialPreserve, acResp.ActivityExpirySettings.PreserveHostActivitiesOnReenrollment)

	// Disable AI features.
	acResp = appConfigResponse{}
	s.DoJSON(
		"PATCH", "/api/latest/fleet/config", json.RawMessage(
			`{
    "server_settings": {
        "ai_features_disabled": true
    }
  }`,
		), http.StatusOK, &acResp,
	)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.ServerSettings.AIFeaturesDisabled)

	// test a change that does clear the agent options (the field is provided but empty).
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": {}
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Equal(t, "{}", string(*acResp.AgentOptions))
	assert.True(t, acResp.MDM.AppleBMTermsExpired)

	// test a change that does modify the agent options.
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"views": {"foo": "bar"}} }
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listActivities, "order_key", "id", "order_direction", "desc")
	require.Greater(t, len(listActivities.Activities), 1)
	require.Equal(t, fleet.ActivityTypeEditedAgentOptions{}.ActivityName(), listActivities.Activities[0].Type)
	require.NotNil(t, listActivities.Activities[0].Details)
	assert.JSONEq(t, `{"global": true, "fleet_id": null, "fleet_name": null, "team_id": null, "team_name": null}`, string(*listActivities.Activities[0].Details))

	// try to set invalid agent options
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"nope": true} }
  }`), http.StatusBadRequest, &acResp)
	// did not update the appconfig
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotContains(t, string(*acResp.AgentOptions), `"nope"`)

	// try to set an invalid agent options logger_tls_endpoint (must start with "/")
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"options": {"logger_tls_endpoint": "not-a-rooted-path"}} }
  }`), http.StatusBadRequest, &acResp)
	// did not update the appconfig
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotContains(t, string(*acResp.AgentOptions), `"not-a-rooted-path"`)

	// try to set a valid agent options logger_tls_endpoint
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"options": {"logger_tls_endpoint": "/rooted-path"}} }
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"/rooted-path"`)

	// force-set invalid agent options
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"nope": true} }
  }`), http.StatusOK, &acResp, "force", "true")
	require.Contains(t, string(*acResp.AgentOptions), `"nope"`)

	// dry-run valid agent options
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"views":{"yep": "ok"}} }
  }`), http.StatusOK, &acResp, "dry_run", "true")
	require.NotContains(t, string(*acResp.AgentOptions), `"yep"`)
	require.Contains(t, string(*acResp.AgentOptions), `"nope"`)

	// dry-run invalid agent options
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"invalid": true} }
  }`), http.StatusBadRequest, &acResp, "dry_run", "true")
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotContains(t, string(*acResp.AgentOptions), `"invalid"`)

	// set valid agent options command-line flag
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "command_line_flags": {"enable_tables":"table1"}}
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"enable_tables": "table1"`)

	// set invalid agent options command-line flag
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "command_line_flags": {"no_such_flag":true}}
  }`), http.StatusBadRequest, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"enable_tables": "table1"`)
	require.NotContains(t, string(*acResp.AgentOptions), `"no_such_flag"`)

	// set invalid value for a valid agent options command-line flag
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "command_line_flags": {"enable_tables":true}}
  }`), http.StatusBadRequest, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"enable_tables": "table1"`)

	// force-set invalid value for a valid agent options command-line flag
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "command_line_flags": {"enable_tables":true}}
  }`), http.StatusOK, &acResp, "force", "true")
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotContains(t, string(*acResp.AgentOptions), `"enable_tables": "table1"`)
	require.Contains(t, string(*acResp.AgentOptions), `"enable_tables": true`)

	// dry-run valid appconfig that uses legacy settings (returns error)
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"host_settings": { "additional_queries": {"foo": "bar"} }
  }`), http.StatusBadRequest, &acResp, "dry_run", "true")
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Nil(t, acResp.Features.AdditionalQueries)

	// without dry-run, the valid appconfig that uses legacy settings is accepted
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"host_settings": { "additional_queries": {"foo": "bar"} }
  }`), http.StatusOK, &acResp, "dry_run", "false")
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp.Features.AdditionalQueries)
	require.Contains(t, string(*acResp.Features.AdditionalQueries), `"foo": "bar"`)

	var verResp versionResponse
	s.DoJSON("GET", "/api/latest/fleet/version", nil, http.StatusOK, &verResp)
	assert.NotEmpty(t, verResp.Branch)

	// get enroll secrets, none yet
	var specResp getEnrollSecretSpecResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/enroll_secret", nil, http.StatusOK, &specResp)
	assert.Empty(t, specResp.Spec.Secrets)

	seenActivitiesIDs := map[uint]struct{}{}
	activityName := fleet.ActivityTypeEditedEnrollSecrets{}.ActivityName()

	// apply spec, one secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: "XYZ"}},
		},
	}, http.StatusOK, &applyResp)

	// adding a new secret should create a new activity entry
	seenActivitiesIDs[s.lastActivityMatches(activityName, "", 0)] = struct{}{}
	require.Len(t, seenActivitiesIDs, 1)

	// applying the same secret again shouldn't create a new activity since we are only interested in mutations
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: "XYZ"}},
		},
	}, http.StatusOK, &applyResp)

	seenActivitiesIDs[s.lastActivityMatches(activityName, "", 0)] = struct{}{}
	require.Len(t, seenActivitiesIDs, 1)

	// apply spec, too many secrets
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: createEnrollSecrets(t, fleet.MaxEnrollSecretsCount+1),
		},
	}, http.StatusUnprocessableEntity, &applyResp)

	// apply spec, empty and whitespace-only secrets are rejected
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: ""}}},
	}, http.StatusUnprocessableEntity, &applyResp)
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: "   "}}},
	}, http.StatusUnprocessableEntity, &applyResp)

	// error conditions should create new activities
	seenActivitiesIDs[s.lastActivityMatches(activityName, "", 0)] = struct{}{}
	require.Len(t, seenActivitiesIDs, 1)

	// get enroll secrets, one
	s.DoJSON("GET", "/api/latest/fleet/spec/enroll_secret", nil, http.StatusOK, &specResp)
	require.Len(t, specResp.Spec.Secrets, 1)
	assert.Equal(t, "XYZ", specResp.Spec.Secrets[0].Secret)

	// remove secret just to prevent affecting other tests
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{},
	}, http.StatusOK, &applyResp)

	// removing the secret should create a new activity entry
	seenActivitiesIDs[s.lastActivityMatches(activityName, "", 0)] = struct{}{}
	require.Len(t, seenActivitiesIDs, 2)

	s.DoJSON("GET", "/api/latest/fleet/spec/enroll_secret", nil, http.StatusOK, &specResp)
	require.Empty(t, specResp.Spec.Secrets)

	// try to update the apple bm terms flag via PATCH /config
	// request is ok but modified value is ignored
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "apple_bm_terms_expired": false }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.AppleBMTermsExpired)

	// try to update the mdm configured flags via PATCH /config
	// request is ok but modified value is ignored
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	  "mdm": { "enabled_and_configured": false, "apple_bm_enabled_and_configured": false }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnabledAndConfigured)
	assert.True(t, acResp.MDM.AppleBMEnabledAndConfigured)

	// set the macos disk encryption field, fails due to license
	res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")

	// legacy config
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": { "enable_disk_encryption": true } }
  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")

	// try to set the apple bm default team, which is premium only
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "apple_bm_default_team": "xyz" }
  }`), http.StatusUnprocessableEntity, &acResp)

	// try to set Okta conditional access settings, which is premium only
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"conditional_access": {
			"okta_idp_id": "https://www.okta.com/saml2/service-provider/test",
			"okta_assertion_consumer_service_url": "https://dev-test.okta.com/sso/saml2/test",
			"okta_audience_uri": "https://www.okta.com/saml2/service-provider/test",
			"okta_certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"
		}
  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")

	// try to set the windows updates, which is premium only
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_updates": {"deadline_days": 1, "grace_period_days": 0} }
  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")

	// try to enable Windows MDM, impossible without the WSTEP certs
	// (only set in mdm integrations tests)
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_enabled_and_configured": true }
  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "Please configure Fleet with a certificate and key pair first.")

	// verify that the Apple BM terms expired flag was never modified
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.AppleBMTermsExpired)

	// set the apple BM terms back to false
	appCfg, err = s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.MDM.AppleBMTermsExpired = false
	appCfg.MDM.AppleBMEnabledAndConfigured = false
	appCfg.MDM.EnabledAndConfigured = false
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)

	// set the macos custom settings fields, fails due to MDM not configured
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": { "macos_settings": { "custom_settings": ["foo", "bar"] } }
	  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "Couldn't update apple_settings because MDM features aren't turned on in Fleet.")

	// test setting the default app config we use for new installs (this check
	// ensures that the default config passes the validation)
	var defAppCfg fleet.AppConfig
	defAppCfg.ApplyDefaultsForNewInstalls()
	// must set org name and server settings
	defAppCfg.OrgInfo.OrgName = acResp.OrgInfo.OrgName
	defAppCfg.ServerSettings.ServerURL = acResp.ServerSettings.ServerURL
	s.DoRaw("PATCH", "/api/latest/fleet/config", jsonMustMarshal(t, defAppCfg), http.StatusOK)

	// turn on GitOps mode, premium only
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"gitops": { "gitops_mode_enabled": true, "repository_url": "" }
	  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")
}

func (s *integrationTestSuite) TestOrgLogoUpload() {
	t := s.T()

	pngImg := image.NewRGBA(image.Rect(0, 0, 1, 1))
	pngImg.Set(0, 0, color.RGBA{R: 0, G: 128, B: 0, A: 255})
	var pngBuf bytes.Buffer
	require.NoError(t, png.Encode(&pngBuf, pngImg))
	pngBytes := pngBuf.Bytes()

	buildLogoBody := func(filename string, content []byte) ([]byte, map[string]string) {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		fw, err := w.CreateFormFile("logo", filename)
		require.NoError(t, err)
		_, err = io.Copy(fw, bytes.NewReader(content))
		require.NoError(t, err)
		require.NoError(t, w.Close())
		return body.Bytes(), map[string]string{
			"Content-Type":  w.FormDataContentType(),
			"Accept":        "application/json",
			"Authorization": "Bearer " + s.token,
		}
	}

	// 1. Upload as admin: 200, AppConfig URL set to the Fleet-hosted serving
	// path, GET returns the bytes back with the right content type.
	body, headers := buildLogoBody("logo.png", pngBytes)
	s.DoRawWithHeaders("PUT", "/api/v1/fleet/logo?mode=light", body, http.StatusOK, headers)

	var acResp appConfigResponse
	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, acResp.OrgInfo.OrgLogoURLLightMode, "/api/latest/fleet/logo")
	require.Contains(t, acResp.OrgInfo.OrgLogoURLLightMode, "mode=light")
	// Deprecated key is in sync.
	require.Equal(t, acResp.OrgInfo.OrgLogoURLLightMode, acResp.OrgInfo.OrgLogoURLLightBackground) //nolint:staticcheck // intentionally asserts the deprecated field stays in sync with its replacement

	res := s.DoRawNoAuth("GET", "/api/latest/fleet/logo?mode=light", nil, http.StatusOK)
	gotBody, err := io.ReadAll(res.Body)
	require.NoError(t, res.Body.Close())
	require.NoError(t, err)
	require.Equal(t, pngBytes, gotBody)
	require.Equal(t, "image/png", res.Header.Get("Content-Type"))

	// 2. Upload a second mode (dark) as admin so the delete-lifecycle assertions
	// at the bottom can confirm modes are independent.
	body, headers = buildLogoBody("dark.png", pngBytes)
	s.DoRawWithHeaders("PUT", "/api/v1/fleet/logo?mode=dark", body, http.StatusOK, headers)

	// 3. Auth: a maintainer is rejected.
	maintainerEmail := "maintainer-logo@example.com"
	maintainerUser := &fleet.User{
		Name:       "Maintainer Logo",
		Email:      maintainerEmail,
		GlobalRole: new(fleet.RoleMaintainer),
	}
	require.NoError(t, maintainerUser.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(t.Context(), maintainerUser)
	require.NoError(t, err)

	s.token = s.getCachedUserToken(maintainerEmail, test.GoodPassword)
	body, headers = buildLogoBody("nope.png", pngBytes)
	s.DoRawWithHeaders("PUT", "/api/v1/fleet/logo?mode=light", body, http.StatusForbidden, headers)
	s.token = s.getTestAdminToken()

	// 4. A non-image payload is rejected at upload time.
	body, headers = buildLogoBody("not-an-image.png", []byte("plain text, definitely not a PNG"))
	s.DoRawWithHeaders("PUT", "/api/v1/fleet/logo?mode=light", body, http.StatusBadRequest, headers)

	// 5. DELETE clears the URL field and the GET endpoint returns 404 for
	// the affected mode while the other mode is unaffected.
	s.Do("DELETE", "/api/v1/fleet/logo", nil, http.StatusOK, "mode", "light")

	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	require.Empty(t, acResp.OrgInfo.OrgLogoURLLightMode)
	require.Empty(t, acResp.OrgInfo.OrgLogoURLLightBackground) //nolint:staticcheck // intentionally asserts the deprecated field stays in sync with its replacement
	require.Contains(t, acResp.OrgInfo.OrgLogoURLDarkMode, "/api/latest/fleet/logo")

	res = s.DoRawNoAuth("GET", "/api/latest/fleet/logo?mode=light", nil, http.StatusNotFound)
	require.NoError(t, res.Body.Close())

	// 6. DELETE is idempotent — a second DELETE for the same mode (now
	// empty) returns 200 with no error.
	s.Do("DELETE", "/api/v1/fleet/logo", nil, http.StatusOK, "mode", "light")

	// 7. DELETE on an external URL (no blob) clears the URL field.
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"org_info": {
			"org_logo_url": "https://placehold.co/100",
			"org_logo_url_dark_mode": "https://placehold.co/100",
			"org_logo_url_light_background": "https://placehold.co/200",
			"org_logo_url_light_mode": "https://placehold.co/200"
		}
	}`), http.StatusOK)

	s.Do("DELETE", "/api/v1/fleet/logo", nil, http.StatusOK, "mode", "dark")
	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	require.Empty(t, acResp.OrgInfo.OrgLogoURLDarkMode)
	require.Empty(t, acResp.OrgInfo.OrgLogoURL) //nolint:staticcheck // intentionally asserts the deprecated field stays in sync with its replacement
	require.Equal(t, "https://placehold.co/200", acResp.OrgInfo.OrgLogoURLLightMode)
	require.Equal(t, "https://placehold.co/200", acResp.OrgInfo.OrgLogoURLLightBackground) //nolint:staticcheck // intentionally asserts the deprecated field stays in sync with its replacement

	// 8. PATCH transitioning a Fleet-hosted URL to an external URL must
	// preserve the external URL and delete the previously-stored blob.
	body, headers = buildLogoBody("light2.png", pngBytes)
	s.DoRawWithHeaders("PUT", "/api/v1/fleet/logo?mode=light", body, http.StatusOK, headers)
	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, acResp.OrgInfo.OrgLogoURLLightMode, "/api/latest/fleet/logo")

	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"org_info": {
			"org_logo_url_light_mode": "https://placehold.co/300",
			"org_logo_url_light_background": "https://placehold.co/300"
		}
	}`), http.StatusOK)
	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	require.Equal(t, "https://placehold.co/300", acResp.OrgInfo.OrgLogoURLLightMode)
	require.Equal(t, "https://placehold.co/300", acResp.OrgInfo.OrgLogoURLLightBackground) //nolint:staticcheck // intentionally asserts the deprecated field stays in sync with its replacement

	// The orphan blob is actually gone: GET /logo?mode=light returns 404.
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/logo?mode=light", nil, http.StatusNotFound)
	require.NoError(t, res.Body.Close())

	// And the cleanup recorded a deleted_org_logo activity.
	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)
	var sawAutoCleanupActivity bool
	for _, a := range activities.Activities {
		if a.Type != "deleted_org_logo" || a.Details == nil {
			continue
		}
		var d struct {
			Mode string `json:"mode"`
		}
		if err := json.Unmarshal(*a.Details, &d); err == nil && d.Mode == string(fleet.OrgLogoModeLight) {
			sawAutoCleanupActivity = true
			break
		}
	}
	assert.True(t, sawAutoCleanupActivity, "auto-cleanup must emit a deleted_org_logo activity for the affected mode")
}
