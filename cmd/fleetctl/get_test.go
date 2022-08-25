package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ghodss/yaml"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var userRoleList = []*fleet.User{
	{
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Now()},
		},
		ID:         42,
		Name:       "Test Name admin1@example.com",
		Email:      "admin1@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	},
	{
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Now()},
		},
		ID:         23,
		Name:       "Test Name2 admin2@example.com",
		Email:      "admin2@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{
					ID:        1,
					CreatedAt: time.Now(),
					Name:      "team1",
					UserCount: 1,
					HostCount: 1,
				},
				Role: fleet.RoleMaintainer,
			},
		},
	},
}

func TestGetUserRoles(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.ListUsersFunc = func(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
		return userRoleList, nil
	}

	expectedText := `+-------------------------------+-------------+
|             USER              | GLOBAL ROLE |
+-------------------------------+-------------+
| Test Name admin1@example.com  | admin       |
+-------------------------------+-------------+
| Test Name2 admin2@example.com |             |
+-------------------------------+-------------+
`
	expectedYaml := `---
apiVersion: v1
kind: user_roles
spec:
  roles:
    admin1@example.com:
      global_role: admin
      teams: null
    admin2@example.com:
      global_role: null
      teams:
      - role: maintainer
        team: team1
`
	expectedJson := `{"kind":"user_roles","apiVersion":"v1","spec":{"roles":{"admin1@example.com":{"global_role":"admin","teams":null},"admin2@example.com":{"global_role":null,"teams":[{"team":"team1","role":"maintainer"}]}}}}
`

	assert.Equal(t, expectedText, runAppForTest(t, []string{"get", "user_roles"}))
	assert.YAMLEq(t, expectedYaml, runAppForTest(t, []string{"get", "user_roles", "--yaml"}))
	assert.JSONEq(t, expectedJson, runAppForTest(t, []string{"get", "user_roles", "--json"}))
}

func TestGetTeams(t *testing.T) {
	var expiredBanner strings.Builder
	fleet.WriteExpiredLicenseBanner(&expiredBanner)
	require.Contains(t, expiredBanner.String(), "Your license for Fleet Premium is about to expire")

	testCases := []struct {
		name                    string
		license                 *fleet.LicenseInfo
		shouldHaveExpiredBanner bool
	}{
		{
			"not expired license",
			&fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)},
			false,
		},
		{
			"expired license",
			&fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(-24 * time.Hour)},
			true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			license := tt.license
			_, ds := runServerWithMockedDS(t, &service.TestServerOpts{License: license})

			agentOpts := json.RawMessage(`{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}}`)
			ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
				created_at, err := time.Parse(time.RFC3339, "1999-03-10T02:45:06.371Z")
				require.NoError(t, err)
				return []*fleet.Team{
					{
						ID:          42,
						CreatedAt:   created_at,
						Name:        "team1",
						Description: "team1 description",
						UserCount:   99,
					},
					{
						ID:          43,
						CreatedAt:   created_at,
						Name:        "team2",
						Description: "team2 description",
						UserCount:   87,
						Config: fleet.TeamConfig{
							AgentOptions: &agentOpts,
						},
					},
				}, nil
			}

			expectedText := `+-----------+-------------------+------------+
| TEAM NAME |    DESCRIPTION    | USER COUNT |
+-----------+-------------------+------------+
| team1     | team1 description |         99 |
+-----------+-------------------+------------+
| team2     | team2 description |         87 |
+-----------+-------------------+------------+
`
			expectedYaml := `---
apiVersion: v1
kind: team
spec:
  team:
    created_at: "1999-03-10T02:45:06.371Z"
    description: team1 description
    host_count: 0
    id: 42
    integrations:
      jira: null
      zendesk: null
    name: team1
    user_count: 99
    webhook_settings:
      failing_policies_webhook:
        destination_url: ""
        enable_failing_policies_webhook: false
        host_batch_size: 0
        policy_ids: null
---
apiVersion: v1
kind: team
spec:
  team:
    agent_options:
      config:
        foo: bar
      overrides:
        platforms:
          darwin:
            foo: override
    created_at: "1999-03-10T02:45:06.371Z"
    description: team2 description
    host_count: 0
    id: 43
    integrations:
      jira: null
      zendesk: null
    name: team2
    user_count: 87
    webhook_settings:
      failing_policies_webhook:
        destination_url: ""
        enable_failing_policies_webhook: false
        host_batch_size: 0
        policy_ids: null
`
			expectedJson := `{"kind":"team","apiVersion":"v1","spec":{"team":{"id":42,"created_at":"1999-03-10T02:45:06.371Z","name":"team1","description":"team1 description","webhook_settings":{"failing_policies_webhook":{"enable_failing_policies_webhook":false,"destination_url":"","policy_ids":null,"host_batch_size":0}},"integrations":{"jira":null,"zendesk":null},"user_count":99,"host_count":0}}}
{"kind":"team","apiVersion":"v1","spec":{"team":{"id":43,"created_at":"1999-03-10T02:45:06.371Z","name":"team2","description":"team2 description","agent_options":{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}},"webhook_settings":{"failing_policies_webhook":{"enable_failing_policies_webhook":false,"destination_url":"","policy_ids":null,"host_batch_size":0}},"integrations":{"jira":null,"zendesk":null},"user_count":87,"host_count":0}}}
`
			if tt.shouldHaveExpiredBanner {
				expectedJson = expiredBanner.String() + expectedJson
				expectedYaml = expiredBanner.String() + expectedYaml
				expectedText = expiredBanner.String() + expectedText
			}

			assert.Equal(t, expectedText, runAppForTest(t, []string{"get", "teams"}))
			assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "teams", "--yaml"}))
			assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "teams", "--json"}))
		})
	}
}

func TestGetTeamsByName(t *testing.T) {
	_, ds := runServerWithMockedDS(t, &service.TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}})

	ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
		require.Equal(t, "test1", opt.MatchQuery)

		created_at, err := time.Parse(time.RFC3339, "1999-03-10T02:45:06.371Z")
		require.NoError(t, err)
		return []*fleet.Team{
			{
				ID:          42,
				CreatedAt:   created_at,
				Name:        "team1",
				Description: "team1 description",
				UserCount:   99,
			},
		}, nil
	}

	expectedText := `+-----------+-------------------+------------+
| TEAM NAME |    DESCRIPTION    | USER COUNT |
+-----------+-------------------+------------+
| team1     | team1 description |         99 |
+-----------+-------------------+------------+
`
	assert.Equal(t, expectedText, runAppForTest(t, []string{"get", "teams", "--name", "test1"}))
}

func TestGetHosts(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	// this func is called when no host is specified i.e. `fleetctl get hosts --json`
	ds.ListHostsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		additional := json.RawMessage(`{"query1": [{"col1": "val", "col2": 42}]}`)
		hosts := []*fleet.Host{
			{
				UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
					CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Time{}},
					UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Time{}},
				},
				HostSoftware:    fleet.HostSoftware{},
				DetailUpdatedAt: time.Time{},
				LabelUpdatedAt:  time.Time{},
				LastEnrolledAt:  time.Time{},
				SeenTime:        time.Time{},
				ComputerName:    "test_host",
				Hostname:        "test_host",
				Additional:      &additional,
			},
			{
				UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
					CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Time{}},
					UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Time{}},
				},
				HostSoftware:    fleet.HostSoftware{},
				DetailUpdatedAt: time.Time{},
				LabelUpdatedAt:  time.Time{},
				LastEnrolledAt:  time.Time{},
				SeenTime:        time.Time{},
				ComputerName:    "test_host2",
				Hostname:        "test_host2",
			},
		}
		return hosts, nil
	}

	// these are run when host is specified `fleetctl get hosts --json test_host`
	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
		require.NotEmpty(t, identifier)
		return &fleet.Host{
			UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
				CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Time{}},
				UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Time{}},
			},
			HostSoftware:    fleet.HostSoftware{},
			DetailUpdatedAt: time.Time{},
			LabelUpdatedAt:  time.Time{},
			LastEnrolledAt:  time.Time{},
			SeenTime:        time.Time{},
			ComputerName:    "test_host",
			Hostname:        "test_host",
		}, nil
	}

	ds.LoadHostSoftwareFunc = func(ctx context.Context, host *fleet.Host, includeCVEScores bool) error {
		return nil
	}
	ds.ListLabelsForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Label, error) {
		return make([]*fleet.Label, 0), nil
	}
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) (packs []*fleet.Pack, err error) {
		return make([]*fleet.Pack, 0), nil
	}
	ds.ListHostBatteriesFunc = func(ctx context.Context, hid uint) (batteries []*fleet.HostBattery, err error) {
		return nil, nil
	}
	defaultPolicyQuery := "select 1 from osquery_info where start_time > 1;"
	ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
		return []*fleet.HostPolicy{
			{
				PolicyData: fleet.PolicyData{
					ID:          1,
					Name:        "query1",
					Query:       defaultPolicyQuery,
					Description: "Some description",
					AuthorID:    ptr.Uint(1),
					AuthorName:  "Alice",
					AuthorEmail: "alice@example.com",
					Resolution:  ptr.String("Some resolution"),
					TeamID:      ptr.Uint(1),
				},
				Response: "passes",
			},
			{
				PolicyData: fleet.PolicyData{
					ID:          2,
					Name:        "query2",
					Query:       defaultPolicyQuery,
					Description: "",
					AuthorID:    ptr.Uint(1),
					AuthorName:  "Alice",
					AuthorEmail: "alice@example.com",
					Resolution:  nil,
					TeamID:      nil,
				},
				Response: "fails",
			},
		}, nil
	}

	expectedText := `+------+------------+----------+-----------------+---------+
| UUID |  HOSTNAME  | PLATFORM | OSQUERY VERSION | STATUS  |
+------+------------+----------+-----------------+---------+
|      | test_host  |          |                 | offline |
+------+------------+----------+-----------------+---------+
|      | test_host2 |          |                 | offline |
+------+------------+----------+-----------------+---------+
`

	jsonPrettify := func(t *testing.T, v string) string {
		var i interface{}
		err := json.Unmarshal([]byte(v), &i)
		require.NoError(t, err)
		indented, err := json.MarshalIndent(i, "", "  ")
		require.NoError(t, err)
		return string(indented)
	}
	yamlPrettify := func(t *testing.T, v string) string {
		var i interface{}
		err := yaml.Unmarshal([]byte(v), &i)
		require.NoError(t, err)
		indented, err := yaml.Marshal(i)
		require.NoError(t, err)
		return string(indented)
	}
	tests := []struct {
		name       string
		goldenFile string
		scanner    func(s string) []string
		prettifier func(t *testing.T, v string) string
		args       []string
	}{
		{
			name:       "get hosts --json",
			goldenFile: "expectedListHostsJson.json",
			scanner: func(s string) []string {
				parts := strings.Split(s, "}\n{")
				return []string{parts[0] + "}", "{" + parts[1]}
			},
			args:       []string{"get", "hosts", "--json"},
			prettifier: jsonPrettify,
		},
		{
			name:       "get hosts --json test_host",
			goldenFile: "expectedHostDetailResponseJson.json",
			scanner:    func(s string) []string { return []string{s} },
			args:       []string{"get", "hosts", "--json", "test_host"},
			prettifier: jsonPrettify,
		},
		{
			name:       "get hosts --yaml",
			goldenFile: "expectedListHostsYaml.yml",
			scanner: func(s string) []string {
				return []string{s}
			},
			args:       []string{"get", "hosts", "--yaml"},
			prettifier: yamlPrettify,
		},
		{
			name:       "get hosts --yaml test_host",
			goldenFile: "expectedHostDetailResponseYaml.yml",
			scanner: func(s string) []string {
				return spec.SplitYaml(s)
			},
			args:       []string{"get", "hosts", "--yaml", "test_host"},
			prettifier: yamlPrettify,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected, err := ioutil.ReadFile(filepath.Join("testdata", tt.goldenFile))
			require.NoError(t, err)
			expectedResults := tt.scanner(string(expected))
			actualResult := tt.scanner(runAppForTest(t, tt.args))
			require.Equal(t, len(expectedResults), len(actualResult))
			for i := range expectedResults {
				require.Equal(t, tt.prettifier(t, expectedResults[i]), tt.prettifier(t, actualResult[i]))
			}
		})
	}

	assert.Equal(t, expectedText, runAppForTest(t, []string{"get", "hosts"}))
}

func TestGetConfig(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			Features:              fleet.Features{EnableHostUsers: true},
			VulnerabilitySettings: fleet.VulnerabilitySettings{DatabasesPath: "/some/path"},
		}, nil
	}

	t.Run("AppConfig", func(t *testing.T) {
		expectedYaml := `---
apiVersion: v1
kind: config
spec:
  fleet_desktop:
    transparency_url: https://fleetdm.com/transparency
  host_expiry_settings:
    host_expiry_enabled: false
    host_expiry_window: 0
  features:
    enable_host_users: true
    enable_software_inventory: false
  integrations:
    jira: null
    zendesk: null
  org_info:
    org_logo_url: ""
    org_name: ""
  server_settings:
    deferred_save_host: false
    enable_analytics: false
    live_query_disabled: false
    server_url: ""
  smtp_settings:
    authentication_method: ""
    authentication_type: ""
    configured: false
    domain: ""
    enable_smtp: false
    enable_ssl_tls: false
    enable_start_tls: false
    password: ""
    port: 0
    sender_address: ""
    server: ""
    user_name: ""
    verify_ssl_certs: false
  sso_settings:
    enable_jit_provisioning: false
    enable_sso: false
    enable_sso_idp_login: false
    entity_id: ""
    idp_image_url: ""
    idp_name: ""
    issuer_uri: ""
    metadata: ""
    metadata_url: ""
  vulnerability_settings:
    databases_path: /some/path
  webhook_settings:
    failing_policies_webhook:
      destination_url: ""
      enable_failing_policies_webhook: false
      host_batch_size: 0
      policy_ids: null
    host_status_webhook:
      days_count: 0
      destination_url: ""
      enable_host_status_webhook: false
      host_percentage: 0
    interval: 0s
    vulnerabilities_webhook:
      destination_url: ""
      enable_vulnerabilities_webhook: false
      host_batch_size: 0
`
		expectedJSON := `
{
  "kind": "config",
  "apiVersion": "v1",
  "spec": {
    "org_info": { "org_name": "", "org_logo_url": "" },
    "server_settings": {
      "server_url": "",
      "live_query_disabled": false,
      "enable_analytics": false,
      "deferred_save_host": false
    },
    "smtp_settings": {
      "enable_smtp": false,
      "configured": false,
      "sender_address": "",
      "server": "",
      "port": 0,
      "authentication_type": "",
      "user_name": "",
      "password": "",
      "enable_ssl_tls": false,
      "authentication_method": "",
      "domain": "",
      "verify_ssl_certs": false,
      "enable_start_tls": false
    },
    "host_expiry_settings": {
      "host_expiry_enabled": false,
      "host_expiry_window": 0
    },
    "features": {
      "enable_host_users": true,
      "enable_software_inventory": false
    },
    "sso_settings": {
      "entity_id": "",
      "issuer_uri": "",
      "idp_image_url": "",
      "metadata": "",
      "metadata_url": "",
      "idp_name": "",
      "enable_jit_provisioning": false,
      "enable_sso": false,
      "enable_sso_idp_login": false
    },
    "fleet_desktop": { "transparency_url": "https://fleetdm.com/transparency" },
    "vulnerability_settings": { "databases_path": "/some/path" },
    "webhook_settings": {
      "host_status_webhook": {
        "enable_host_status_webhook": false,
        "destination_url": "",
        "host_percentage": 0,
        "days_count": 0
      },
      "failing_policies_webhook": {
        "enable_failing_policies_webhook": false,
        "destination_url": "",
        "policy_ids": null,
        "host_batch_size": 0
      },
      "vulnerabilities_webhook": {
        "enable_vulnerabilities_webhook": false,
        "destination_url": "",
        "host_batch_size": 0
      },
      "interval": "0s"
    },
    "integrations": { "jira": null, "zendesk": null }
  }
}
`

		assert.YAMLEq(t, expectedYaml, runAppForTest(t, []string{"get", "config"}))
		assert.YAMLEq(t, expectedYaml, runAppForTest(t, []string{"get", "config", "--yaml"}))
		assert.JSONEq(t, expectedJSON, runAppForTest(t, []string{"get", "config", "--json"}))
	})

	t.Run("IncludeServerConfig", func(t *testing.T) {
		expectedYaml := `---
apiVersion: v1
kind: config
spec:
  fleet_desktop:
    transparency_url: https://fleetdm.com/transparency
  host_expiry_settings:
    host_expiry_enabled: false
    host_expiry_window: 0
  features:
    enable_host_users: true
    enable_software_inventory: false
  integrations:
    jira: null
    zendesk: null
  license:
    expiration: "0001-01-01T00:00:00Z"
    tier: free
  logging:
    debug: true
    json: false
    result:
      config:
        enable_log_compression: false
        enable_log_rotation: false
        result_log_file: /dev/null
        status_log_file: /dev/null
      plugin: filesystem
    status:
      config:
        enable_log_compression: false
        enable_log_rotation: false
        result_log_file: /dev/null
        status_log_file: /dev/null
      plugin: filesystem
  org_info:
    org_logo_url: ""
    org_name: ""
  server_settings:
    deferred_save_host: false
    enable_analytics: false
    live_query_disabled: false
    server_url: ""
  smtp_settings:
    authentication_method: ""
    authentication_type: ""
    configured: false
    domain: ""
    enable_smtp: false
    enable_ssl_tls: false
    enable_start_tls: false
    password: ""
    port: 0
    sender_address: ""
    server: ""
    user_name: ""
    verify_ssl_certs: false
  sso_settings:
    enable_jit_provisioning: false
    enable_sso: false
    enable_sso_idp_login: false
    entity_id: ""
    idp_image_url: ""
    idp_name: ""
    issuer_uri: ""
    metadata: ""
    metadata_url: ""
  update_interval:
    osquery_detail: 1h0m0s
    osquery_policy: 1h0m0s
  vulnerabilities:
    cpe_database_url: ""
    current_instance_checks: ""
    cve_feed_prefix_url: ""
    databases_path: ""
    disable_data_sync: false
    periodicity: 0s
    recent_vulnerability_max_age: 0s
  vulnerability_settings:
    databases_path: /some/path
  webhook_settings:
    failing_policies_webhook:
      destination_url: ""
      enable_failing_policies_webhook: false
      host_batch_size: 0
      policy_ids: null
    host_status_webhook:
      days_count: 0
      destination_url: ""
      enable_host_status_webhook: false
      host_percentage: 0
    interval: 0s
    vulnerabilities_webhook:
      destination_url: ""
      enable_vulnerabilities_webhook: false
      host_batch_size: 0
`
		expectedJSON := `
{
  "kind": "config",
  "apiVersion": "v1",
  "spec": {
    "org_info": { "org_name": "", "org_logo_url": "" },
    "server_settings": {
      "server_url": "",
      "live_query_disabled": false,
      "enable_analytics": false,
      "deferred_save_host": false
    },
    "smtp_settings": {
      "enable_smtp": false,
      "configured": false,
      "sender_address": "",
      "server": "",
      "port": 0,
      "authentication_type": "",
      "user_name": "",
      "password": "",
      "enable_ssl_tls": false,
      "authentication_method": "",
      "domain": "",
      "verify_ssl_certs": false,
      "enable_start_tls": false
    },
    "host_expiry_settings": {
      "host_expiry_enabled": false,
      "host_expiry_window": 0
    },
    "features": {
      "enable_host_users": true,
      "enable_software_inventory": false
    },
    "sso_settings": {
      "enable_jit_provisioning": false,
      "entity_id": "",
      "issuer_uri": "",
      "idp_image_url": "",
      "metadata": "",
      "metadata_url": "",
      "idp_name": "",
      "enable_sso": false,
      "enable_sso_idp_login": false
    },
    "fleet_desktop": { "transparency_url": "https://fleetdm.com/transparency" },
    "vulnerability_settings": { "databases_path": "/some/path" },
    "webhook_settings": {
      "host_status_webhook": {
        "enable_host_status_webhook": false,
        "destination_url": "",
        "host_percentage": 0,
        "days_count": 0
      },
      "failing_policies_webhook": {
        "enable_failing_policies_webhook": false,
        "destination_url": "",
        "policy_ids": null,
        "host_batch_size": 0
      },
      "vulnerabilities_webhook": {
        "enable_vulnerabilities_webhook": false,
        "destination_url": "",
        "host_batch_size": 0
      },
      "interval": "0s"
    },
    "integrations": { "jira": null, "zendesk": null },
    "update_interval": {
      "osquery_detail": "1h0m0s",
      "osquery_policy": "1h0m0s"
    },
    "vulnerabilities": {
      "databases_path": "",
      "periodicity": "0s",
      "cpe_database_url": "",
      "cve_feed_prefix_url": "",
      "current_instance_checks": "",
      "disable_data_sync": false,
      "recent_vulnerability_max_age": "0s"
    },
    "license": { "tier": "free", "expiration": "0001-01-01T00:00:00Z" },
    "logging": {
      "debug": true,
      "json": false,
      "result": {
        "plugin": "filesystem",
        "config": {
          "enable_log_compression": false,
          "enable_log_rotation": false,
          "result_log_file": "/dev/null",
          "status_log_file": "/dev/null"
        }
      },
      "status": {
        "plugin": "filesystem",
        "config": {
          "enable_log_compression": false,
          "enable_log_rotation": false,
          "result_log_file": "/dev/null",
          "status_log_file": "/dev/null"
        }
      }
    }
  }
}
`

		assert.YAMLEq(t, expectedYaml, runAppForTest(t, []string{"get", "config", "--include-server-config"}))
		assert.YAMLEq(t, expectedYaml, runAppForTest(t, []string{"get", "config", "--include-server-config", "--yaml"}))
		require.JSONEq(t, expectedJSON, runAppForTest(t, []string{"get", "config", "--include-server-config", "--json"}))
	})
}

func TestGetSoftware(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	foo001 := fleet.Software{
		Name: "foo", Version: "0.0.1", Source: "chrome_extensions", GenerateCPE: "somecpe",
		Vulnerabilities: fleet.Vulnerabilities{
			{CVE: "cve-321-432-543", DetailsLink: "https://nvd.nist.gov/vuln/detail/cve-321-432-543"},
			{CVE: "cve-333-444-555", DetailsLink: "https://nvd.nist.gov/vuln/detail/cve-333-444-555"},
		},
	}
	foo002 := fleet.Software{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"}
	foo003 := fleet.Software{Name: "foo", Version: "0.0.3", Source: "chrome_extensions", GenerateCPE: "someothercpewithoutvulns"}
	bar003 := fleet.Software{Name: "bar", Version: "0.0.3", Source: "deb_packages", BundleIdentifier: "bundle"}

	var gotTeamID *uint

	ds.ListSoftwareFunc = func(ctx context.Context, opt fleet.SoftwareListOptions) ([]fleet.Software, error) {
		gotTeamID = opt.TeamID
		return []fleet.Software{foo001, foo002, foo003, bar003}, nil
	}

	expected := `+------+---------+-------------------+--------------------------+-----------+
| NAME | VERSION |      SOURCE       |           CPE            | # OF CVES |
+------+---------+-------------------+--------------------------+-----------+
| foo  | 0.0.1   | chrome_extensions | somecpe                  |         2 |
+------+---------+-------------------+--------------------------+-----------+
| foo  | 0.0.2   | chrome_extensions |                          |         0 |
+------+---------+-------------------+--------------------------+-----------+
| foo  | 0.0.3   | chrome_extensions | someothercpewithoutvulns |         0 |
+------+---------+-------------------+--------------------------+-----------+
| bar  | 0.0.3   | deb_packages      |                          |         0 |
+------+---------+-------------------+--------------------------+-----------+
`

	expectedYaml := `---
apiVersion: "1"
kind: software
spec:
- generated_cpe: somecpe
  id: 0
  name: foo
  source: chrome_extensions
  version: 0.0.1
  vulnerabilities:
  - cve: cve-321-432-543
    details_link: https://nvd.nist.gov/vuln/detail/cve-321-432-543
  - cve: cve-333-444-555
    details_link: https://nvd.nist.gov/vuln/detail/cve-333-444-555
- generated_cpe: ""
  id: 0
  name: foo
  source: chrome_extensions
  version: 0.0.2
  vulnerabilities: null
- generated_cpe: someothercpewithoutvulns
  id: 0
  name: foo
  source: chrome_extensions
  version: 0.0.3
  vulnerabilities: null
- bundle_identifier: bundle
  generated_cpe: ""
  id: 0
  name: bar
  source: deb_packages
  version: 0.0.3
  vulnerabilities: null
`

	expectedJson := `
{
  "kind": "software",
  "apiVersion": "1",
  "spec": [
    {
      "id": 0,
      "name": "foo",
      "version": "0.0.1",
      "source": "chrome_extensions",
      "generated_cpe": "somecpe",
      "vulnerabilities": [
        {
          "cve": "cve-321-432-543",
          "details_link": "https://nvd.nist.gov/vuln/detail/cve-321-432-543"
        },
        {
          "cve": "cve-333-444-555",
          "details_link": "https://nvd.nist.gov/vuln/detail/cve-333-444-555"
        }
      ]
    },
    {
      "id": 0,
      "name": "foo",
      "version": "0.0.2",
      "source": "chrome_extensions",
      "generated_cpe": "",
      "vulnerabilities": null
    },
    {
      "id": 0,
      "name": "foo",
      "version": "0.0.3",
      "source": "chrome_extensions",
      "generated_cpe": "someothercpewithoutvulns",
      "vulnerabilities": null
    },
    {
      "id": 0,
      "name": "bar",
      "version": "0.0.3",
      "bundle_identifier": "bundle",
      "source": "deb_packages",
      "generated_cpe": "",
      "vulnerabilities": null
    }
  ]
}
`

	assert.Equal(t, expected, runAppForTest(t, []string{"get", "software"}))
	assert.YAMLEq(t, expectedYaml, runAppForTest(t, []string{"get", "software", "--yaml"}))
	assert.JSONEq(t, expectedJson, runAppForTest(t, []string{"get", "software", "--json"}))

	runAppForTest(t, []string{"get", "software", "--json", "--team", "999"})
	require.NotNil(t, gotTeamID)
	assert.Equal(t, uint(999), *gotTeamID)
}

func TestGetLabels(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.GetLabelSpecsFunc = func(ctx context.Context) ([]*fleet.LabelSpec, error) {
		return []*fleet.LabelSpec{
			{
				ID:          32,
				Name:        "label1",
				Description: "some description",
				Query:       "select 1;",
				Platform:    "windows",
			},
			{
				ID:          33,
				Name:        "label2",
				Description: "some other description",
				Query:       "select 42;",
				Platform:    "linux",
			},
		}, nil
	}

	expected := `+--------+----------+------------------------+------------+
|  NAME  | PLATFORM |      DESCRIPTION       |   QUERY    |
+--------+----------+------------------------+------------+
| label1 | windows  | some description       | select 1;  |
+--------+----------+------------------------+------------+
| label2 | linux    | some other description | select 42; |
+--------+----------+------------------------+------------+
`
	expectedYaml := `---
apiVersion: v1
kind: label
spec:
  description: some description
  id: 32
  label_membership_type: dynamic
  name: label1
  platform: windows
  query: select 1;
---
apiVersion: v1
kind: label
spec:
  description: some other description
  id: 33
  label_membership_type: dynamic
  name: label2
  platform: linux
  query: select 42;
`
	expectedJson := `{"kind":"label","apiVersion":"v1","spec":{"id":32,"name":"label1","description":"some description","query":"select 1;","platform":"windows","label_membership_type":"dynamic"}}
{"kind":"label","apiVersion":"v1","spec":{"id":33,"name":"label2","description":"some other description","query":"select 42;","platform":"linux","label_membership_type":"dynamic"}}
`

	assert.Equal(t, expected, runAppForTest(t, []string{"get", "labels"}))
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "labels", "--yaml"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "labels", "--json"}))
}

func TestGetLabel(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.GetLabelSpecFunc = func(ctx context.Context, name string) (*fleet.LabelSpec, error) {
		if name != "label1" {
			return nil, nil
		}
		return &fleet.LabelSpec{
			ID:          32,
			Name:        "label1",
			Description: "some description",
			Query:       "select 1;",
			Platform:    "windows",
		}, nil
	}

	expectedYaml := `---
apiVersion: v1
kind: label
spec:
  description: some description
  id: 32
  label_membership_type: dynamic
  name: label1
  platform: windows
  query: select 1;
`
	expectedJson := `{"kind":"label","apiVersion":"v1","spec":{"id":32,"name":"label1","description":"some description","query":"select 1;","platform":"windows","label_membership_type":"dynamic"}}
`

	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "label", "label1"}))
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "label", "--yaml", "label1"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "label", "--json", "label1"}))
}

func TestGetEnrollmentSecrets(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.GetEnrollSecretsFunc = func(ctx context.Context, teamID *uint) ([]*fleet.EnrollSecret, error) {
		return []*fleet.EnrollSecret{
			{
				Secret: "abcd",
				TeamID: nil,
			},
			{
				Secret: "efgh",
				TeamID: nil,
			},
		}, nil
	}

	expectedYaml := `---
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
  - created_at: "0001-01-01T00:00:00Z"
    secret: abcd
  - created_at: "0001-01-01T00:00:00Z"
    secret: efgh
`
	expectedJson := `{"kind":"enroll_secret","apiVersion":"v1","spec":{"secrets":[{"secret":"abcd","created_at":"0001-01-01T00:00:00Z"},{"secret":"efgh","created_at":"0001-01-01T00:00:00Z"}]}}
`

	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "enroll_secrets"}))
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "enroll_secrets", "--yaml"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "enroll_secrets", "--json"}))
}

func TestGetPacks(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.GetPackSpecsFunc = func(ctx context.Context) ([]*fleet.PackSpec, error) {
		return []*fleet.PackSpec{
			{
				ID:          7,
				Name:        "pack1",
				Description: "some desc",
				Platform:    "darwin",
				Disabled:    false,
			},
		}, nil
	}

	expected := `+-------+----------+-------------+----------+
| NAME  | PLATFORM | DESCRIPTION | DISABLED |
+-------+----------+-------------+----------+
| pack1 | darwin   | some desc   | false    |
+-------+----------+-------------+----------+
`
	expectedYaml := `---
apiVersion: v1
kind: pack
spec:
  description: some desc
  disabled: false
  id: 7
  name: pack1
  platform: darwin
  targets:
    labels: null
    teams: null
`
	expectedJson := `
{
  "kind": "pack",
  "apiVersion": "v1",
  "spec": {
    "id": 7,
    "name": "pack1",
    "description": "some desc",
    "platform": "darwin",
    "disabled": false,
    "targets": {
      "labels": null,
      "teams": null
    }
  }
}
`

	assert.Equal(t, expected, runAppForTest(t, []string{"get", "packs"}))
	assert.YAMLEq(t, expectedYaml, runAppForTest(t, []string{"get", "packs", "--yaml"}))
	assert.JSONEq(t, expectedJson, runAppForTest(t, []string{"get", "packs", "--json"}))
}

func TestGetPack(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.PackByNameFunc = func(ctx context.Context, name string, opts ...fleet.OptionalArg) (*fleet.Pack, bool, error) {
		if name != "pack1" {
			return nil, false, nil
		}
		return &fleet.Pack{
			ID:          7,
			Name:        "pack1",
			Description: "some desc",
			Platform:    "darwin",
			Disabled:    false,
		}, true, nil
	}
	ds.GetPackSpecFunc = func(ctx context.Context, name string) (*fleet.PackSpec, error) {
		if name != "pack1" {
			return nil, nil
		}
		return &fleet.PackSpec{
			ID:          7,
			Name:        "pack1",
			Description: "some desc",
			Platform:    "darwin",
			Disabled:    false,
		}, nil
	}

	expectedYaml := `---
apiVersion: v1
kind: pack
spec:
  description: some desc
  disabled: false
  id: 7
  name: pack1
  platform: darwin
  targets:
    labels: null
    teams: null
`
	expectedJson := `
{
  "kind": "pack",
  "apiVersion": "v1",
  "spec": {
    "id": 7,
    "name": "pack1",
    "description": "some desc",
    "platform": "darwin",
    "disabled": false,
    "targets": {
      "labels": null,
      "teams": null
    }
  }
}
`

	assert.YAMLEq(t, expectedYaml, runAppForTest(t, []string{"get", "packs", "pack1"}))
	assert.YAMLEq(t, expectedYaml, runAppForTest(t, []string{"get", "packs", "--yaml", "pack1"}))
	assert.JSONEq(t, expectedJson, runAppForTest(t, []string{"get", "packs", "--json", "pack1"}))
}

func TestGetQueries(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.ListQueriesFunc = func(ctx context.Context, opt fleet.ListQueryOptions) ([]*fleet.Query, error) {
		return []*fleet.Query{
			{
				ID:             33,
				Name:           "query1",
				Description:    "some desc",
				Query:          "select 1;",
				Saved:          false,
				ObserverCanRun: false,
			},
			{
				ID:             12,
				Name:           "query2",
				Description:    "some desc 2",
				Query:          "select 2;",
				Saved:          true,
				ObserverCanRun: false,
			},
		}, nil
	}

	expected := `+--------+-------------+-----------+
|  NAME  | DESCRIPTION |   QUERY   |
+--------+-------------+-----------+
| query1 | some desc   | select 1; |
+--------+-------------+-----------+
| query2 | some desc 2 | select 2; |
+--------+-------------+-----------+
`
	expectedYaml := `---
apiVersion: v1
kind: query
spec:
  description: some desc
  name: query1
  query: select 1;
---
apiVersion: v1
kind: query
spec:
  description: some desc 2
  name: query2
  query: select 2;
`
	expectedJson := `{"kind":"query","apiVersion":"v1","spec":{"name":"query1","description":"some desc","query":"select 1;"}}
{"kind":"query","apiVersion":"v1","spec":{"name":"query2","description":"some desc 2","query":"select 2;"}}
`

	assert.Equal(t, expected, runAppForTest(t, []string{"get", "queries"}))
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "queries", "--yaml"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "queries", "--json"}))
}

func TestGetQuery(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.QueryByNameFunc = func(ctx context.Context, name string, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		if name != "query1" {
			return nil, nil
		}
		return &fleet.Query{
			ID:             33,
			Name:           "query1",
			Description:    "some desc",
			Query:          "select 1;",
			Saved:          false,
			ObserverCanRun: false,
		}, nil
	}

	expectedYaml := `---
apiVersion: v1
kind: query
spec:
  description: some desc
  name: query1
  query: select 1;
`
	expectedJson := `{"kind":"query","apiVersion":"v1","spec":{"name":"query1","description":"some desc","query":"select 1;"}}
`

	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "query", "query1"}))
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "query", "--yaml", "query1"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "query", "--json", "query1"}))
}

func TestEnrichedAppConfig(t *testing.T) {
	t.Run("deprecated fields", func(t *testing.T) {
		resp := []byte(`
      {
        "org_info": {
          "org_name": "Fleet for osquery",
          "org_logo_url": ""
        },
        "server_settings": {
          "server_url": "https://localhost:8412",
          "live_query_disabled": false,
          "enable_analytics": false,
          "deferred_save_host": false
        },
        "smtp_settings": {
          "enable_smtp": false,
          "configured": false,
          "sender_address": "",
          "server": "",
          "port": 587,
          "authentication_type": "authtype_username_password",
          "user_name": "",
          "password": "",
          "enable_ssl_tls": true,
          "authentication_method": "authmethod_plain",
          "domain": "",
          "verify_ssl_certs": true,
          "enable_start_tls": true
        },
        "host_expiry_settings": {
          "host_expiry_enabled": false,
          "host_expiry_window": 0
        },
        "host_settings": {
          "enable_host_users": true,
          "enable_software_inventory": true
        },
        "agent_options": {
          "config": {
            "options": {
              "logger_plugin": "tls",
              "pack_delimiter": "/",
              "logger_tls_period": 10,
              "distributed_plugin": "tls",
              "disable_distributed": false,
              "logger_tls_endpoint": "/api/osquery/log",
              "distributed_interval": 10,
              "distributed_tls_max_attempts": 3
            },
            "decorators": {
              "load": [
                "SELECT uuid AS host_uuid FROM system_info;",
                "SELECT hostname AS hostname FROM system_info;"
              ]
            }
          },
          "overrides": {}
        },
        "sso_settings": {
          "entity_id": "",
          "issuer_uri": "",
          "idp_image_url": "",
          "metadata": "",
          "metadata_url": "",
          "idp_name": "",
          "enable_sso": false,
          "enable_sso_idp_login": false,
          "enable_jit_provisioning": false
        },
        "fleet_desktop": {
          "transparency_url": "https://fleetdm.com/transparency"
        },
        "vulnerability_settings": {
          "databases_path": ""
        },
        "webhook_settings": {
          "host_status_webhook": {
            "enable_host_status_webhook": false,
            "destination_url": "",
            "host_percentage": 0,
            "days_count": 0
          },
          "failing_policies_webhook": {
            "enable_failing_policies_webhook": false,
            "destination_url": "",
            "policy_ids": null,
            "host_batch_size": 0
          },
          "vulnerabilities_webhook": {
            "enable_vulnerabilities_webhook": false,
            "destination_url": "",
            "host_batch_size": 0
          },
          "interval": "24h0m0s"
        },
        "integrations": {
          "jira": null,
          "zendesk": null
        },
        "update_interval": {
          "osquery_detail": 3600000000000,
          "osquery_policy": 3600000000000
        },
        "vulnerabilities": {
          "databases_path": "/vulndb",
          "periodicity": 300000000000,
          "cpe_database_url": "",
          "cve_feed_prefix_url": "",
          "current_instance_checks": "yes",
          "disable_data_sync": false,
          "recent_vulnerability_max_age": 2592000000000000
        },
        "license": {
          "tier": "free",
          "expiration": "0001-01-01T00:00:00Z"
        },
        "logging": {
          "debug": true,
          "json": true,
          "result": {
            "plugin": "filesystem",
            "config": {
              "status_log_file": "/logs/osqueryd.status.log",
              "result_log_file": "/logs/osqueryd.results.log",
              "enable_log_rotation": false,
              "enable_log_compression": false
            }
          },
          "status": {
            "plugin": "filesystem",
            "config": {
              "status_log_file": "/logs/osqueryd.status.log",
              "result_log_file": "/logs/osqueryd.results.log",
              "enable_log_rotation": false,
              "enable_log_compression": false
            }
          }
        }
      }
    `)

		var enriched fleet.EnrichedAppConfig
		err := json.Unmarshal(resp, &enriched)
		require.NoError(t, err)
		require.NotNil(t, enriched.Vulnerabilities)
		require.Equal(t, "yes", enriched.Vulnerabilities.CurrentInstanceChecks)
		require.True(t, enriched.Features.EnableSoftwareInventory)
		require.Equal(t, "free", enriched.License.Tier)
		require.Equal(t, "filesystem", enriched.Logging.Status.Plugin)
	})
}
