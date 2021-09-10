package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
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
	expiredBanner := "Your license for Fleet Premium is about to expire. If you’d like to renew or have questions about downgrading, please navigate to https://github.com/fleetdm/fleet/blob/main/docs/1-Using-Fleet/10-Teams.md#expired_license and contact us for help."
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
			server, ds := runServerWithMockedDS(t, service.TestServerOpts{License: license})
			defer server.Close()

			agentOpts := json.RawMessage(`{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}}`)
			ds.ListTeamsFunc = func(filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
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
				expectedJson = expiredBanner + "\n" + expectedJson
				expectedYaml = expiredBanner + "\n" + expectedYaml
				expectedText = expiredBanner + "\n" + expectedText
			}

			assert.Equal(t, expectedText, runAppForTest(t, []string{"get", "teams"}))
			assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "teams", "--yaml"}))
			assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "teams", "--json"}))
		})
	}
}

func TestGetHosts(t *testing.T) {
	server, ds := runServerWithMockedDS(t)
	defer server.Close()

	// this func is called when no host is specified i.e. `fleetctl get hosts --json`
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
	ds.HostByIdentifierFunc = func(identifier string) (*fleet.Host, error) {
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
			Hostname:        "test_host"}, nil
	}

	ds.LoadHostSoftwareFunc = func(host *fleet.Host) error {
		return nil
	}
	ds.ListLabelsForHostFunc = func(hid uint) ([]*fleet.Label, error) {
		return make([]*fleet.Label, 0), nil
	}
	ds.ListPacksForHostFunc = func(hid uint) (packs []*fleet.Pack, err error) {
		return make([]*fleet.Pack, 0), nil
	}

	expectedText := `+------+------------+----------+-----------------+--------+
| UUID |  HOSTNAME  | PLATFORM | OSQUERY VERSION | STATUS |
+------+------------+----------+-----------------+--------+
|      | test_host  |          |                 | mia    |
+------+------------+----------+-----------------+--------+
|      | test_host2 |          |                 | mia    |
+------+------------+----------+-----------------+--------+
`

	tests := []struct {
		name        string
		goldenFile  string
		unmarshaler func(data []byte, v interface{}) error
		scanner     func(s string) []string
		args        []string
	}{
		{
			name:        "get hosts --json",
			goldenFile:  "expectedListHostsJson.json",
			unmarshaler: json.Unmarshal,
			scanner: func(s string) []string {
				var parts []string
				scanner := bufio.NewScanner(bytes.NewBufferString(s))
				for scanner.Scan() {
					parts = append(parts, scanner.Text())
				}
				return parts
			},
			args: []string{"get", "hosts", "--json"},
		},
		{
			name:        "get hosts --json test_host",
			goldenFile:  "expectedHostDetailResponseJson.json",
			unmarshaler: json.Unmarshal,
			scanner: func(s string) []string {
				return []string{s}
			},
			args: []string{"get", "hosts", "--json", "test_host"},
		},
		{
			name:        "get hosts --yaml",
			goldenFile:  "expectedListHostsYaml.yml",
			unmarshaler: yaml.Unmarshal,
			scanner: func(s string) []string {
				return []string{s}
			},
			args: []string{"get", "hosts", "--yaml"},
		},
		{
			name:        "get hosts --yaml test_host",
			goldenFile:  "expectedHostDetailResponseYaml.yml",
			unmarshaler: yaml.Unmarshal,
			scanner: func(s string) []string {
				return splitYaml(s)
			},
			args: []string{"get", "hosts", "--yaml", "test_host"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected, err := ioutil.ReadFile(filepath.Join("testdata", tt.goldenFile))
			require.NoError(t, err)
			expectedResults := tt.scanner(string(expected))
			expectedSpecs := make([]specGeneric, len(expectedResults))
			for i, result := range expectedResults {
				var got specGeneric
				require.NoError(t, tt.unmarshaler([]byte(result), &got))
				expectedSpecs[i] = got
			}
			actualResult := tt.scanner(runAppForTest(t, tt.args))
			actualSpecs := make([]specGeneric, len(actualResult))
			for i, result := range actualResult {
				var spec specGeneric
				require.NoError(t, tt.unmarshaler([]byte(result), &spec))
				actualSpecs[i] = spec
			}
			require.Equal(t, expectedSpecs, actualSpecs)
		})
	}

	assert.Equal(t, expectedText, runAppForTest(t, []string{"get", "hosts"}))
}

func TestGetConfig(t *testing.T) {
	server, ds := runServerWithMockedDS(t)
	defer server.Close()

	ds.AppConfigFunc = func() (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			HostSettings:          fleet.HostSettings{EnableHostUsers: true},
			VulnerabilitySettings: fleet.VulnerabilitySettings{DatabasesPath: "/some/path"},
		}, nil
	}

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
	expectedJson := `{"kind":"config","apiVersion":"v1","spec":{"org_info":{"org_name":"","org_logo_url":""},"server_settings":{"server_url":"","live_query_disabled":false,"enable_analytics":false},"smtp_settings":{"enable_smtp":false,"configured":false,"sender_address":"","server":"","port":0,"authentication_type":"","user_name":"","password":"","enable_ssl_tls":false,"authentication_method":"","domain":"","verify_ssl_certs":false,"enable_start_tls":false},"host_expiry_settings":{"host_expiry_enabled":false,"host_expiry_window":0},"host_settings":{"enable_host_users":true,"enable_software_inventory":false},"sso_settings":{"entity_id":"","issuer_uri":"","idp_image_url":"","metadata":"","metadata_url":"","idp_name":"","enable_sso":false,"enable_sso_idp_login":false},"vulnerability_settings":{"databases_path":"/some/path"},"webhook_settings":{"host_status_webhook":{"enable_host_status_webhook":false,"destination_url":"","host_percentage":0,"days_count":0},"interval":"0s"}}}
`

	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "config"}))
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "config", "--yaml"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "config", "--json"}))
}

func TestGetSoftawre(t *testing.T) {
	server, ds := runServerWithMockedDS(t)
	defer server.Close()

	foo001 := fleet.Software{
		Name: "foo", Version: "0.0.1", Source: "chrome_extensions", GenerateCPE: "somecpe",
		Vulnerabilities: fleet.VulnerabilitiesSlice{
			{"cve-321-432-543", "https://nvd.nist.gov/vuln/detail/cve-321-432-543"},
			{"cve-333-444-555", "https://nvd.nist.gov/vuln/detail/cve-333-444-555"},
		},
	}
	foo002 := fleet.Software{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"}
	foo003 := fleet.Software{Name: "foo", Version: "0.0.3", Source: "chrome_extensions", GenerateCPE: "someothercpewithoutvulns"}
	bar003 := fleet.Software{Name: "bar", Version: "0.0.3", Source: "deb_packages"}

	var gotTeamID *uint

	ds.ListSoftwareFunc = func(ctx context.Context, teamId *uint, opt fleet.ListOptions) ([]fleet.Software, error) {
		gotTeamID = teamId
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
- generated_cpe: ""
  id: 0
  name: bar
  source: deb_packages
  version: 0.0.3
  vulnerabilities: null
`
	expectedJson := `{"kind":"software","apiVersion":"1","spec":[{"id":0,"name":"foo","version":"0.0.1","source":"chrome_extensions","generated_cpe":"somecpe","vulnerabilities":[{"cve":"cve-321-432-543","details_link":"https://nvd.nist.gov/vuln/detail/cve-321-432-543"},{"cve":"cve-333-444-555","details_link":"https://nvd.nist.gov/vuln/detail/cve-333-444-555"}]},{"id":0,"name":"foo","version":"0.0.2","source":"chrome_extensions","generated_cpe":"","vulnerabilities":null},{"id":0,"name":"foo","version":"0.0.3","source":"chrome_extensions","generated_cpe":"someothercpewithoutvulns","vulnerabilities":null},{"id":0,"name":"bar","version":"0.0.3","source":"deb_packages","generated_cpe":"","vulnerabilities":null}]}
`

	assert.Equal(t, expected, runAppForTest(t, []string{"get", "software"}))
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "software", "--yaml"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "software", "--json"}))

	runAppForTest(t, []string{"get", "software", "--json", "--team", "999"})
	require.NotNil(t, gotTeamID)
	assert.Equal(t, uint(999), *gotTeamID)
}
