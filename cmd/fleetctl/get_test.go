package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var userRoleList = []*fleet.User{
	&fleet.User{
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Now()},
		},
		ID:         42,
		Name:       "Test Name admin1@example.com",
		Email:      "admin1@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	},
	&fleet.User{
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Now()},
		},
		ID:         23,
		Name:       "Test Name2 admin2@example.com",
		Email:      "admin2@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			fleet.UserTeam{
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
	server, ds := runServerWithMockedDS(t)
	defer server.Close()

	ds.ListUsersFunc = func(opt fleet.UserListOptions) ([]*fleet.User, error) {
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
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "user_roles", "--yaml"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "user_roles", "--json"}))
}

func TestGetTeams(t *testing.T) {
	server, ds := runServerWithMockedDS(t, service.TestServerOpts{Tier: fleet.TierBasic})
	defer server.Close()

	agentOpts := json.RawMessage(`{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}}`)
	ds.ListTeamsFunc = func(filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
		created_at, err := time.Parse(time.RFC3339, "1999-03-10T02:45:06.371Z")
		require.NoError(t, err)
		return []*fleet.Team{
			&fleet.Team{
				ID:          42,
				CreatedAt:   created_at,
				Name:        "team1",
				Description: "team1 description",
				UserCount:   99,
			},
			&fleet.Team{
				ID:           43,
				CreatedAt:    created_at,
				Name:         "team2",
				Description:  "team2 description",
				UserCount:    87,
				AgentOptions: &agentOpts,
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
    agent_options: null
    created_at: "1999-03-10T02:45:06.371Z"
    description: team1 description
    host_count: 0
    id: 42
    name: team1
    user_count: 99
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
    name: team2
    user_count: 87
`
	expectedJson := `{"kind":"team","apiVersion":"v1","spec":{"team":{"id":42,"created_at":"1999-03-10T02:45:06.371Z","name":"team1","description":"team1 description","agent_options":null,"user_count":99,"host_count":0}}}
{"kind":"team","apiVersion":"v1","spec":{"team":{"id":43,"created_at":"1999-03-10T02:45:06.371Z","name":"team2","description":"team2 description","agent_options":{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}},"user_count":87,"host_count":0}}}
`

	assert.Equal(t, expectedText, runAppForTest(t, []string{"get", "teams"}))
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "teams", "--yaml"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "teams", "--json"}))
}

func TestGetHosts(t *testing.T) {
	server, ds := runServerWithMockedDS(t)
	defer server.Close()

	ds.ListHostsFunc = func(filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
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
			},
		}
		return hosts, nil
	}

	expectedText := `+------+-----------+----------+-----------------+--------+
| UUID | HOSTNAME  | PLATFORM | OSQUERY VERSION | STATUS |
+------+-----------+----------+-----------------+--------+
|      | test_host |          |                 | mia    |
+------+-----------+----------+-----------------+--------+
`

	expectedYaml := `---
apiVersion: v1
kind: host
spec:
  build: ""
  code_name: ""
  computer_name: test_host
  config_tls_refresh: 0
  cpu_brand: ""
  cpu_logical_cores: 0
  cpu_physical_cores: 0
  cpu_subtype: ""
  cpu_type: ""
  created_at: "0001-01-01T00:00:00Z"
  detail_updated_at: "0001-01-01T00:00:00Z"
  display_text: test_host
  distributed_interval: 0
  hardware_model: ""
  hardware_serial: ""
  hardware_vendor: ""
  hardware_version: ""
  hostname: test_host
  id: 0
  label_updated_at: "0001-01-01T00:00:00Z"
  last_enrolled_at: "0001-01-01T00:00:00Z"
  logger_tls_period: 0
  memory: 0
  os_version: ""
  osquery_version: ""
  pack_stats: null
  platform: ""
  platform_like: ""
  primary_ip: ""
  primary_mac: ""
  refetch_requested: false
  seen_time: "0001-01-01T00:00:00Z"
  status: mia
  team_id: null
  team_name: null
  updated_at: "0001-01-01T00:00:00Z"
  uptime: 0
  uuid: ""
`
	expectedJson := "{\"kind\":\"host\",\"apiVersion\":\"v1\",\"spec\":{\"created_at\":\"0001-01-01T00:00:00Z\",\"updated_at\":\"0001-01-01T00:00:00Z\",\"id\":0,\"detail_updated_at\":\"0001-01-01T00:00:00Z\",\"label_updated_at\":\"0001-01-01T00:00:00Z\",\"last_enrolled_at\":\"0001-01-01T00:00:00Z\",\"seen_time\":\"0001-01-01T00:00:00Z\",\"refetch_requested\":false,\"hostname\":\"test_host\",\"uuid\":\"\",\"platform\":\"\",\"osquery_version\":\"\",\"os_version\":\"\",\"build\":\"\",\"platform_like\":\"\",\"code_name\":\"\",\"uptime\":0,\"memory\":0,\"cpu_type\":\"\",\"cpu_subtype\":\"\",\"cpu_brand\":\"\",\"cpu_physical_cores\":0,\"cpu_logical_cores\":0,\"hardware_vendor\":\"\",\"hardware_model\":\"\",\"hardware_version\":\"\",\"hardware_serial\":\"\",\"computer_name\":\"test_host\",\"primary_ip\":\"\",\"primary_mac\":\"\",\"distributed_interval\":0,\"config_tls_refresh\":0,\"logger_tls_period\":0,\"team_id\":null,\"pack_stats\":null,\"team_name\":null,\"status\":\"mia\",\"display_text\":\"test_host\"}}\n"

	assert.Equal(t, expectedText, runAppForTest(t, []string{"get", "hosts"}))
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "hosts", "--yaml"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "hosts", "--json"}))
}

func TestGetConfig(t *testing.T) {
	server, ds := runServerWithMockedDS(t)
	defer server.Close()

	ds.AppConfigFunc = func() (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			EnableHostUsers:            true,
			VulnerabilityDatabasesPath: ptr.String("/some/path"),
		}, nil
	}

	expectedYaml := `---
apiVersion: v1
kind: config
spec:
  agent_options: null
  host_expiry_settings:
    host_expiry_enabled: false
    host_expiry_window: 0
  host_settings:
    enable_host_users: true
    enable_software_inventory: false
  org_info:
    org_logo_url: ""
    org_name: ""
  server_settings:
    enable_analytics: false
    live_query_disabled: false
    server_url: ""
  smtp_settings:
    authentication_method: authmethod_plain
    authentication_type: authtype_username_password
    configured: false
    domain: ""
    enable_smtp: false
    enable_ssl_tls: false
    enable_start_tls: false
    password: '********'
    port: 0
    sender_address: ""
    server: ""
    user_name: ""
    verify_ssl_certs: false
  sso_settings:
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
`
	expectedJson := `{"kind":"config","apiVersion":"v1","spec":{"org_info":{"org_name":"","org_logo_url":""},"server_settings":{"server_url":"","live_query_disabled":false,"enable_analytics":false},"smtp_settings":{"enable_smtp":false,"configured":false,"sender_address":"","server":"","port":0,"authentication_type":"authtype_username_password","user_name":"","password":"********","enable_ssl_tls":false,"authentication_method":"authmethod_plain","domain":"","verify_ssl_certs":false,"enable_start_tls":false},"host_expiry_settings":{"host_expiry_enabled":false,"host_expiry_window":0},"host_settings":{"enable_host_users":true,"enable_software_inventory":false},"agent_options":null,"sso_settings":{"entity_id":"","issuer_uri":"","idp_image_url":"","metadata":"","metadata_url":"","idp_name":"","enable_sso":false,"enable_sso_idp_login":false},"vulnerability_settings":{"databases_path":"/some/path"}}}
`

	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "config"}))
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "config", "--yaml"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "config", "--json"}))
}
