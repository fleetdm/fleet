package orbitconfig_notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

type captureNotifier struct {
	calls []string
}

func (c *captureNotifier) NotifyAgentsForLiveQuery(ctx context.Context, hostIDs []uint, campaignID uint) error {
	return errors.New("unexpected")
}

func (c *captureNotifier) NotifyOrbitConfigHosts(ctx context.Context, hostIDs []uint, reason string) error {
	c.calls = append(c.calls, fmt.Sprintf("hosts %v %s", hostIDs, reason))
	return nil
}

func (c *captureNotifier) NotifyOrbitConfigScope(ctx context.Context, scope fleet.AgentNotificationScope, reason string) error {
	c.calls = append(c.calls, fmt.Sprintf("scope %s %s", scope, reason))
	return nil
}

func newTestDatastore() (*mock.Store, *captureNotifier, *Datastore) {
	ds := new(mock.Store)
	notifier := &captureNotifier{}
	return ds, notifier, New(ds, notifier, slog.New(slog.DiscardHandler))
}

// hookCase exercises one wrapped method. setup configures the mock with the
// state before the write; run performs the write with a changed (or, when
// unchanged is set, identical) value.
type hookCase struct {
	name string
	// want is the notification expected after a changing write.
	want string
	run  func(ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error
	// alwaysNotifies is set for methods that notify without comparing.
	alwaysNotifies bool
}

var hookCases = []hookCase{
	{
		name: "SaveAppConfig",
		want: "scope global settings",
		run: func(ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			before := &fleet.AppConfig{AgentOptions: new(json.RawMessage(`{"script_execution_timeout":300}`))}
			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return before, nil }
			ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error { return writeErr }
			after := &fleet.AppConfig{AgentOptions: new(json.RawMessage(`{"script_execution_timeout":300}`))}
			if !unchanged {
				after.MDM.WindowsEnabledAndConfigured = true
			}
			return d.SaveAppConfig(context.Background(), after)
		},
	},
	{
		name: "SaveTeam",
		want: "scope team:3 settings",
		run: func(ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
				return &fleet.TeamLite{ID: tid, Config: fleet.TeamConfigLite{AgentOptions: new(json.RawMessage(`{}`))}}, nil
			}
			ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) { return team, writeErr }
			team := &fleet.Team{ID: 3, Config: fleet.TeamConfig{AgentOptions: new(json.RawMessage(`{}`))}}
			if !unchanged {
				team.Config.AgentOptions = new(json.RawMessage(`{"command_line_flags":{"verbose":true}}`))
			}
			_, err := d.SaveTeam(context.Background(), team)
			return err
		},
	},
	{
		name: "SaveDefaultTeamConfig",
		want: "scope team:0 settings",
		run: func(ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.DefaultTeamConfigFunc = func(ctx context.Context) (*fleet.TeamConfig, error) {
				return &fleet.TeamConfig{}, nil
			}
			ds.SaveDefaultTeamConfigFunc = func(ctx context.Context, config *fleet.TeamConfig) error { return writeErr }
			config := &fleet.TeamConfig{}
			if !unchanged {
				config.MDM.EnableDiskEncryption = true
			}
			return d.SaveDefaultTeamConfig(context.Background(), config)
		},
	},
	{
		name:           "DeleteTeam",
		want:           "scope team:0 settings",
		alwaysNotifies: true,
		run: func(ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.DeleteTeamFunc = func(ctx context.Context, tid uint) error { return writeErr }
			return d.DeleteTeam(context.Background(), 3)
		},
	},
	{
		name:           "AddHostsToTeam",
		want:           "hosts [1 2] settings",
		alwaysNotifies: true,
		run: func(ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.AddHostsToTeamFunc = func(ctx context.Context, params *fleet.AddHostsToTeamParams) error { return writeErr }
			return d.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(new(uint(3)), []uint{1, 2}))
		},
	},
}

func TestHooks(t *testing.T) {
	for _, c := range hookCases {
		t.Run(c.name, func(t *testing.T) {
			t.Run("changed", func(t *testing.T) {
				ds, notifier, d := newTestDatastore()
				require.NoError(t, c.run(ds, d, false, nil))
				require.Equal(t, []string{c.want}, notifier.calls)
			})

			t.Run("unchanged", func(t *testing.T) {
				ds, notifier, d := newTestDatastore()
				require.NoError(t, c.run(ds, d, true, nil))
				if c.alwaysNotifies {
					require.Equal(t, []string{c.want}, notifier.calls)
				} else {
					require.Empty(t, notifier.calls)
				}
			})

			t.Run("write fails", func(t *testing.T) {
				ds, notifier, d := newTestDatastore()
				require.Error(t, c.run(ds, d, false, errors.New("boom")))
				require.Empty(t, notifier.calls)
			})
		})
	}
}

func TestSaveAppConfigUnreadableBeforeNotifies(t *testing.T) {
	ds, notifier, d := newTestDatastore()
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return nil, errors.New("boom") }
	ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error { return nil }
	require.NoError(t, d.SaveAppConfig(context.Background(), &fleet.AppConfig{}))
	require.Equal(t, []string{"scope global settings"}, notifier.calls)
}

// TestEveryWrappedMethodHasAHookCase requires each method declared by the
// decorator to have a hook case proving it notifies (see the etag_invalidate
// test of the same name for why unwrapped methods can't be detected).
func TestEveryWrappedMethodHasAHookCase(t *testing.T) {
	covered := map[string]struct{}{}
	for _, c := range hookCases {
		covered[c.name] = struct{}{}
	}

	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	declRE := regexp.MustCompile(`(?m)^func \(d \*Datastore\) ([A-Z]\w*)\(`)
	var declared, missing []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		require.NoError(t, err)
		for _, m := range declRE.FindAllStringSubmatch(string(src), -1) {
			declared = append(declared, m[1])
			if _, ok := covered[m[1]]; !ok {
				missing = append(missing, m[1])
			}
		}
	}
	require.NotEmpty(t, declared, "found no declared decorator methods — did the receiver name change?")
	sort.Strings(missing)
	require.Empty(t, missing, "add a hook case for these wrapped methods: %v", missing)
}
