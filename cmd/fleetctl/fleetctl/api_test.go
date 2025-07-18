package fleetctl

import (
	"context"
	"fmt"
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
      "updated_at": "%s"
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
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "passing_host_count": 0,
    "failing_host_count": 0,
    "host_count_updated_at": null
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
				"\\u003ctestdata/does-not-exist.sql"),
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

			b, err := RunAppNoChecks(args)
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
