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
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "user_roles", "--yaml"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "user_roles", "--json"}))
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
			_, ds := runServerWithMockedDS(t, service.TestServerOpts{License: license})

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

	ds.LoadHostSoftwareFunc = func(ctx context.Context, host *fleet.Host) error {
		return nil
	}
	ds.ListLabelsForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Label, error) {
		return make([]*fleet.Label, 0), nil
	}
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) (packs []*fleet.Pack, err error) {
		return make([]*fleet.Pack, 0), nil
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

	expectedText := `+------+------------+----------+-----------------+--------+
| UUID |  HOSTNAME  | PLATFORM | OSQUERY VERSION | STATUS |
+------+------------+----------+-----------------+--------+
|      | test_host  |          |                 | mia    |
+------+------------+----------+-----------------+--------+
|      | test_host2 |          |                 | mia    |
+------+------------+----------+-----------------+--------+
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
				return splitYaml(s)
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
			HostSettings:          fleet.HostSettings{EnableHostUsers: true},
			VulnerabilitySettings: fleet.VulnerabilitySettings{DatabasesPath: "/some/path"},
		}, nil
	}

	t.Run("AppConfig", func(t *testing.T) {
		expectedYaml := `---
apiVersion: v1
kind: config
spec:
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
    host_status_webhook:
      days_count: 0
      destination_url: ""
      enable_host_status_webhook: false
      host_percentage: 0
    interval: 0s
`
		expectedJson := `{"kind":"config","apiVersion":"v1","spec":{"org_info":{"org_name":"","org_logo_url":""},"server_settings":{"server_url":"","live_query_disabled":false,"enable_analytics":false,"deferred_save_host":false},"smtp_settings":{"enable_smtp":false,"configured":false,"sender_address":"","server":"","port":0,"authentication_type":"","user_name":"","password":"","enable_ssl_tls":false,"authentication_method":"","domain":"","verify_ssl_certs":false,"enable_start_tls":false},"host_expiry_settings":{"host_expiry_enabled":false,"host_expiry_window":0},"host_settings":{"enable_host_users":true,"enable_software_inventory":false},"sso_settings":{"entity_id":"","issuer_uri":"","idp_image_url":"","metadata":"","metadata_url":"","idp_name":"","enable_sso":false,"enable_sso_idp_login":false},"vulnerability_settings":{"databases_path":"/some/path"},"webhook_settings":{"host_status_webhook":{"enable_host_status_webhook":false,"destination_url":"","host_percentage":0,"days_count":0},"interval":"0s"}}}
`

		assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "config"}))
		assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "config", "--yaml"}))
		assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "config", "--json"}))
	})

	t.Run("IncludeServerConfig", func(t *testing.T) {
		expectedYaml := `---
apiVersion: v1
kind: config
spec:
  host_expiry_settings:
    host_expiry_enabled: false
    host_expiry_window: 0
  host_settings:
    enable_host_users: true
    enable_software_inventory: false
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
    enable_sso: false
    enable_sso_idp_login: false
    entity_id: ""
    idp_image_url: ""
    idp_name: ""
    issuer_uri: ""
    metadata: ""
    metadata_url: ""
  update_interval:
    osquery_detail: 3600000000000
    osquery_policy: 3600000000000
  vulnerabilities:
    cpe_database_url: ""
    current_instance_checks: ""
    cve_feed_prefix_url: ""
    databases_path: ""
    disable_data_sync: false
    periodicity: 0
  vulnerability_settings:
    databases_path: /some/path
  webhook_settings:
    host_status_webhook:
      days_count: 0
      destination_url: ""
      enable_host_status_webhook: false
      host_percentage: 0
    interval: 0s
`
		expectedJson := `{"kind":"config","apiVersion":"v1","spec":{"org_info":{"org_name":"","org_logo_url":""},"server_settings":{"server_url":"","live_query_disabled":false,"enable_analytics":false,"deferred_save_host":false},"smtp_settings":{"enable_smtp":false,"configured":false,"sender_address":"","server":"","port":0,"authentication_type":"","user_name":"","password":"","enable_ssl_tls":false,"authentication_method":"","domain":"","verify_ssl_certs":false,"enable_start_tls":false},"host_expiry_settings":{"host_expiry_enabled":false,"host_expiry_window":0},"host_settings":{"enable_host_users":true,"enable_software_inventory":false},"sso_settings":{"entity_id":"","issuer_uri":"","idp_image_url":"","metadata":"","metadata_url":"","idp_name":"","enable_sso":false,"enable_sso_idp_login":false},"vulnerability_settings":{"databases_path":"/some/path"},"webhook_settings":{"host_status_webhook":{"enable_host_status_webhook":false,"destination_url":"","host_percentage":0,"days_count":0},"interval":"0s"},"update_interval":{"osquery_detail":3600000000000,"osquery_policy":3600000000000},"vulnerabilities":{"databases_path":"","periodicity":0,"cpe_database_url":"","cve_feed_prefix_url":"","current_instance_checks":"","disable_data_sync":false},"license":{"tier":"free","expiration":"0001-01-01T00:00:00Z"},"logging":{"debug":true,"json":false,"result":{"plugin":"filesystem","config":{"enable_log_compression":false,"enable_log_rotation":false,"result_log_file":"/dev/null","status_log_file":"/dev/null"}},"status":{"plugin":"filesystem","config":{"enable_log_compression":false,"enable_log_rotation":false,"result_log_file":"/dev/null","status_log_file":"/dev/null"}}}}}
`

		assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "config", "--include-server-config"}))
		assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "config", "--include-server-config", "--yaml"}))
		assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "config", "--include-server-config", "--json"}))
	})
}

func TestGetSoftawre(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	foo001 := fleet.Software{
		Name: "foo", Version: "0.0.1", Source: "chrome_extensions", GenerateCPE: "somecpe",
		Vulnerabilities: fleet.VulnerabilitiesSlice{
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
	expectedJson := `{"kind":"software","apiVersion":"1","spec":[{"id":0,"name":"foo","version":"0.0.1","source":"chrome_extensions","generated_cpe":"somecpe","vulnerabilities":[{"cve":"cve-321-432-543","details_link":"https://nvd.nist.gov/vuln/detail/cve-321-432-543"},{"cve":"cve-333-444-555","details_link":"https://nvd.nist.gov/vuln/detail/cve-333-444-555"}]},{"id":0,"name":"foo","version":"0.0.2","source":"chrome_extensions","generated_cpe":"","vulnerabilities":null},{"id":0,"name":"foo","version":"0.0.3","source":"chrome_extensions","generated_cpe":"someothercpewithoutvulns","vulnerabilities":null},{"id":0,"name":"bar","version":"0.0.3","bundle_identifier":"bundle","source":"deb_packages","generated_cpe":"","vulnerabilities":null}]}
`

	assert.Equal(t, expected, runAppForTest(t, []string{"get", "software"}))
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "software", "--yaml"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "software", "--json"}))

	runAppForTest(t, []string{"get", "software", "--json", "--team", "999"})
	require.NotNil(t, gotTeamID)
	assert.Equal(t, uint(999), *gotTeamID)
}
