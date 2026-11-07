package fleetctl

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/require"
)

// TestRunApiCommand checks that the usage of `api` command works as expected
func TestRunApiCommand(t *testing.T) {
	cfg := config.TestConfig()
	_, ds := testing_utils.RunServerWithMockedDS(t, &service.TestServerOpts{
		License:     &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)},
		FleetConfig: &cfg,
	})

	created_at, err := time.Parse(time.RFC3339, "1999-03-10T02:45:06.371Z")
	require.NoError(t, err)

	type testCase struct {
		name         string
		args         []string
		expectOutput string
		expectErrMsg string
	}

	expectedScripts := `{
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  },
  "scripts": [
    {
      "id": 23,
      "team_id": null,
      "name": "get_my_device_page.sh",
      "created_at": "%s",
      "updated_at": "%s",
      "fleet_id": null
    }
  ]
}
`

	expectedEmptyScripts := `{
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  },
  "scripts": []
}
`

	expectedNewPolicy := `{
  "policy": {
    "id": 0,
    "name": "Test Policy",
    "query": "%s",
    "critical": false,
    "description": "",
    "author_id": 1,
    "author_name": "",
    "author_email": "",
    "team_id": null,
    "resolution": "",
    "platform": "darwin,windows,linux,chrome",
    "calendar_events_enabled": false,
    "conditional_access_enabled": false,
    "hidden": false,
    "type": "dynamic",
    "continuous_automations_enabled": false,
    "patch_when_closed": false,
    "notify_before_patching": false,
    "patch_deadline": false,
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "passing_host_count": 0,
    "failing_host_count": 0,
    "host_count_updated_at": null,
    "fleet_id": null
  }
}
`

	expectedNewScript := `{
  "script_id": 1
}
`

	cases := []testCase{
		{
			name: "get scripts",
			args: []string{"scripts"},
			expectOutput: fmt.Sprintf(
				expectedScripts,
				created_at.Format(time.RFC3339Nano),
				created_at.Format(time.RFC3339Nano)),
		},
		{
			name: "get /scripts",
			args: []string{"/scripts"},
			expectOutput: fmt.Sprintf(
				expectedScripts,
				created_at.Format(time.RFC3339Nano),
				created_at.Format(time.RFC3339Nano)),
		},
		{
			name: "get scripts full path",
			args: []string{"/api/v1/fleet/scripts"},
			expectOutput: fmt.Sprintf(
				expectedScripts,
				created_at.Format(time.RFC3339Nano),
				created_at.Format(time.RFC3339Nano)),
		},
		{
			name: "get scripts full path missing /",
			args: []string{"api/v1/fleet/scripts"},
			expectOutput: fmt.Sprintf(
				expectedScripts,
				created_at.Format(time.RFC3339Nano),
				created_at.Format(time.RFC3339Nano)),
		},
		{
			name:         "get scripts team",
			args:         []string{"-F", "team_id=1", "scripts"},
			expectOutput: expectedEmptyScripts,
		},
		{
			name:         "get scripts team no cache",
			args:         []string{"-H", "cache-control:no-cache", "-F", "team_id=1", "scripts"},
			expectOutput: expectedEmptyScripts,
		},
		{
			name:         "get typo",
			args:         []string{"vresion"},
			expectErrMsg: "Got non 2XX return of 404",
		},
		{
			name:         "flags after args",
			args:         []string{"scripts", "-F", "team_id=1"},
			expectErrMsg: "extra arguments: -F team_id=1",
		},
		{
			name: "create policy",
			args: []string{
				"-X", "POST",
				"-F", "name=Test Policy",
				"-F", "query=<testdata/test-policy-query.sql",
				"-F", "platform=darwin,windows,linux,chrome",
				"-F", "critical=false",
				"-F", "description=",
				"-F", "resolution=",
				"/api/latest/fleet/policies",
			},
			expectOutput: fmt.Sprintf(
				expectedNewPolicy,
				"SELECT 1;"),
		},
		{
			name: "create policy, missing input file",
			args: []string{
				"-X", "POST",
				"-F", "name=Test Policy",
				"-F", "query=<testdata/does-not-exist.sql",
				"-F", "platform=darwin,windows,linux,chrome",
				"-F", "critical=false",
				"-F", "description=",
				"-F", "resolution=",
				"/api/latest/fleet/policies",
			},
			expectOutput: fmt.Sprintf(
				expectedNewPolicy,
				"<testdata/does-not-exist.sql"),
		},
		{
			name: "upload script",
			args: []string{
				"-X", "POST",
				"-F", "script=@testdata/testscript.sh",
				"-F", "team_id=0",
				"/api/latest/fleet/scripts",
			},
			expectOutput: expectedNewScript,
		},
		{
			name: "args after uri",
			args: []string{
				"/api/latest/fleet/foo",
				"-X", "DELETE",
			},
			expectErrMsg: "Ensure any flags are before the URL",
		},
	}

	setupDS := func(t *testing.T, c testCase) {
		ds.ListScriptsFunc = func(ctx context.Context, teamID *uint, opt fleet.ListOptions) ([]*fleet.Script, *fleet.PaginationMetadata, error) {
			if teamID == nil {
				ret := []*fleet.Script{
					{
						ID:        23,
						Name:      "get_my_device_page.sh",
						CreatedAt: created_at,
						UpdatedAt: created_at,
					},
				}
				page := fleet.PaginationMetadata{}
				return ret, &page, nil
			}
			return []*fleet.Script{}, &fleet.PaginationMetadata{}, nil
		}
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupDS(t, c)
			args := []string{"api"}

			args = append(args, c.args...)

			b, err := runAppNoChecks(args)
			if c.expectErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.expectErrMsg)
			} else {
				require.NoError(t, err)
			}
			if c.expectOutput != "" {
				out := b.String()
				require.NoError(t, err)
				require.NotEmpty(t, out)
				require.Equal(t, c.expectOutput, out)
			} else {
				require.Empty(t, b.String())
			}
		})
	}
}

func TestGetToken(t *testing.T) {
	writeTestConfig := func(t *testing.T, token string) string {
		configPath := filepath.Join(t.TempDir(), "config")
		require.NoError(t, writeConfig(configPath, configFile{
			Contexts: map[string]Context{"default": {Token: token}},
		}))
		return configPath
	}

	t.Run("env var unset uses config token", func(t *testing.T) {
		t.Setenv(apiTokenEnvVar, "")
		token, err := getToken(writeTestConfig(t, "config-token"), "default")
		require.NoError(t, err)
		require.Equal(t, "config-token", token)
	})

	t.Run("env var set and config token empty", func(t *testing.T) {
		t.Setenv(apiTokenEnvVar, "env-token")
		token, err := getToken(writeTestConfig(t, ""), "default")
		require.NoError(t, err)
		require.Equal(t, "env-token", token)
	})

	t.Run("env var takes precedence over config token", func(t *testing.T) {
		t.Setenv(apiTokenEnvVar, "env-token")
		token, err := getToken(writeTestConfig(t, "config-token"), "default")
		require.NoError(t, err)
		require.Equal(t, "env-token", token)
	})
}

func TestClientFromCLIUsesTokenEnvVar(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config")

	var gotAuth atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth.Store(r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	t.Setenv("FLEET_SERVER_ADDRESS", srv.URL)

	runAppForTest(t, []string{"config", "set", "--config", configFile, "--token", "config-token", "--address", srv.URL})

	t.Setenv(apiTokenEnvVar, "env-token")
	runAppNoChecks([]string{"get", "packs", "--config", configFile}) //nolint:errcheck
	require.Equal(t, "Bearer env-token", gotAuth.Load())
}

func TestConfigSetTokenWarnsWhenEnvVarOverrides(t *testing.T) {
	cases := []struct {
		desc     string
		envToken string
		wantWarn bool
	}{
		{desc: "env var unset", envToken: "", wantWarn: false},
		{desc: "env var matches saved token", envToken: "abc", wantWarn: false},
		{desc: "env var differs from saved token", envToken: "other", wantWarn: true},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			t.Setenv(apiTokenEnvVar, c.envToken)
			configFile := filepath.Join(t.TempDir(), "config")

			var errBuf bytes.Buffer
			_, err := runWithErrWriter([]string{"config", "set", "--config", configFile, "--token", "abc"}, &errBuf)
			require.NoError(t, err)
			if c.wantWarn {
				require.Contains(t, errBuf.String(), apiTokenEnvVar+" is set and will be used instead of the token saved in the config")
			} else {
				require.NotContains(t, errBuf.String(), apiTokenEnvVar)
			}
		})
	}
}
