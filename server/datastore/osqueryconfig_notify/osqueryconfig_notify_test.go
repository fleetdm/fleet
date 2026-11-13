package osqueryconfig_notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
	return errors.New("unexpected")
}

func (c *captureNotifier) NotifyOrbitConfigScope(ctx context.Context, scope fleet.AgentNotificationScope, reason string) error {
	return errors.New("unexpected")
}

func (c *captureNotifier) NotifyOsqueryConfigHosts(ctx context.Context, hostIDs []uint, reason string) error {
	c.calls = append(c.calls, fmt.Sprintf("hosts %v %s", hostIDs, reason))
	return nil
}

func (c *captureNotifier) NotifyOsqueryConfigScope(ctx context.Context, scope fleet.AgentNotificationScope, reason string) error {
	c.calls = append(c.calls, fmt.Sprintf("scope %s %s", scope, reason))
	return nil
}

func newTestDatastore() (*mock.Store, *captureNotifier, *Datastore) {
	ds := new(mock.Store)
	notifier := &captureNotifier{}
	return ds, notifier, New(ds, notifier, slog.New(slog.DiscardHandler))
}

// mockReportsDigest makes the reports digest change once write is called,
// unless unchanged is set.
func mockReportsDigest(ds *mock.Store, unchanged bool) (write func()) {
	written := false
	ds.ScheduledQueriesDigestFunc = func(ctx context.Context, teamID *uint) (string, error) {
		if written && !unchanged {
			return "after", nil
		}
		return "before", nil
	}
	return func() { written = true }
}

// hookCase exercises one wrapped method. run performs the write with a
// changed (or, when unchanged is set, identical) value.
type hookCase struct {
	name string
	// want is the notification expected after a changing write.
	want string
	run  func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error
	// alwaysNotifies is set for methods that notify without comparing.
	alwaysNotifies bool
	// notifiesOnError is set for methods whose failed write may have
	// partially committed.
	notifiesOnError bool
}

var hookCases = []hookCase{
	{
		name: "SaveAppConfig",
		want: "scope global settings",
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			before := &fleet.AppConfig{AgentOptions: new(json.RawMessage(`{"config":{"options":{"pack_delimiter":"/"}}}`))}
			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return before, nil }
			ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error { return writeErr }
			// Formatting differences don't count as a change.
			after := &fleet.AppConfig{AgentOptions: new(json.RawMessage(`{ "config": {"options": {"pack_delimiter": "/"}} }`))}
			// Settings the osquery config doesn't read don't count either.
			after.MDM.WindowsEnabledAndConfigured = true
			if !unchanged {
				after.AgentOptions = new(json.RawMessage(`{"config":{"options":{"pack_delimiter":"_"}}}`))
			}
			return d.SaveAppConfig(ctx, after)
		},
	},
	{
		name: "SaveTeam",
		want: "scope team:3 settings",
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
				return &fleet.TeamLite{ID: tid, Config: fleet.TeamConfigLite{AgentOptions: new(json.RawMessage(`{}`))}}, nil
			}
			ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) { return team, writeErr }
			team := &fleet.Team{ID: 3, Config: fleet.TeamConfig{AgentOptions: new(json.RawMessage(`{}`))}}
			if !unchanged {
				team.Config.AgentOptions = new(json.RawMessage(`{"config":{"options":{"distributed_interval":30}}}`))
			}
			_, err := d.SaveTeam(ctx, team)
			return err
		},
	},
	{
		name:           "DeleteTeam",
		want:           "scope team:0 settings",
		alwaysNotifies: true,
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.DeleteTeamFunc = func(ctx context.Context, tid uint) error { return writeErr }
			return d.DeleteTeam(ctx, 3)
		},
	},
	{
		name:            "AddHostsToTeam",
		want:            "hosts [1 2] settings",
		alwaysNotifies:  true,
		notifiesOnError: true,
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.AddHostsToTeamFunc = func(ctx context.Context, params *fleet.AddHostsToTeamParams) error { return writeErr }
			return d.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(new(uint(3)), []uint{1, 2}))
		},
	},
	{
		name:            "ApplyQueries",
		want:            "scope team:3 reports",
		notifiesOnError: true,
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			write := mockReportsDigest(ds, unchanged)
			ds.ApplyQueriesFunc = func(ctx context.Context, authorID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]struct{}) error {
				write()
				return writeErr
			}
			return d.ApplyQueries(ctx, 1, []*fleet.Query{{Name: "a", TeamID: new(uint(3))}, {Name: "b", TeamID: new(uint(3))}}, nil)
		},
	},
	{
		name: "NewQuery",
		want: "scope global reports",
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			write := mockReportsDigest(ds, unchanged)
			ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
				if writeErr == nil {
					write()
				}
				return query, writeErr
			}
			_, err := d.NewQuery(ctx, &fleet.Query{Name: "a", Saved: true, Interval: 60})
			return err
		},
	},
	{
		name: "SaveQuery",
		want: "scope team:3 reports",
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			write := mockReportsDigest(ds, unchanged)
			ds.SaveQueryFunc = func(ctx context.Context, query *fleet.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
				if writeErr == nil {
					write()
				}
				return writeErr
			}
			return d.SaveQuery(ctx, &fleet.Query{Name: "a", TeamID: new(uint(3)), Saved: true, Interval: 60}, false, false)
		},
	},
	{
		name: "DeleteQuery",
		want: "scope team:3 reports",
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			write := mockReportsDigest(ds, unchanged)
			ds.DeleteQueryFunc = func(ctx context.Context, teamID *uint, name string) error {
				if writeErr == nil {
					write()
				}
				return writeErr
			}
			return d.DeleteQuery(ctx, new(uint(3)), "a")
		},
	},
	{
		name: "DeleteQueries",
		want: "scope global reports",
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.DeleteQueriesFunc = func(ctx context.Context, ids []uint) (uint, error) {
				if unchanged {
					return 0, writeErr
				}
				return uint(len(ids)), writeErr
			}
			_, err := d.DeleteQueries(ctx, []uint{1, 2})
			return err
		},
	},
	{
		name:           "ApplyPackSpecs",
		want:           "scope global reports",
		alwaysNotifies: true,
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.ApplyPackSpecsFunc = func(ctx context.Context, specs []*fleet.PackSpec) error { return writeErr }
			return d.ApplyPackSpecs(ctx, nil)
		},
	},
	{
		name:           "NewPack",
		want:           "scope global reports",
		alwaysNotifies: true,
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.NewPackFunc = func(ctx context.Context, pack *fleet.Pack, opts ...fleet.OptionalArg) (*fleet.Pack, error) {
				return pack, writeErr
			}
			_, err := d.NewPack(ctx, &fleet.Pack{})
			return err
		},
	},
	{
		name:           "SavePack",
		want:           "scope global reports",
		alwaysNotifies: true,
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.SavePackFunc = func(ctx context.Context, pack *fleet.Pack) error { return writeErr }
			return d.SavePack(ctx, &fleet.Pack{})
		},
	},
	{
		name:           "DeletePack",
		want:           "scope global reports",
		alwaysNotifies: true,
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.DeletePackFunc = func(ctx context.Context, name string) error { return writeErr }
			return d.DeletePack(ctx, "p")
		},
	},
	{
		name:           "NewScheduledQuery",
		want:           "scope global reports",
		alwaysNotifies: true,
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.NewScheduledQueryFunc = func(ctx context.Context, sq *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
				return sq, writeErr
			}
			_, err := d.NewScheduledQuery(ctx, &fleet.ScheduledQuery{})
			return err
		},
	},
	{
		name:           "SaveScheduledQuery",
		want:           "scope global reports",
		alwaysNotifies: true,
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.SaveScheduledQueryFunc = func(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
				return sq, writeErr
			}
			_, err := d.SaveScheduledQuery(ctx, &fleet.ScheduledQuery{})
			return err
		},
	},
	{
		name:           "DeleteScheduledQuery",
		want:           "scope global reports",
		alwaysNotifies: true,
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.DeleteScheduledQueryFunc = func(ctx context.Context, id uint) error { return writeErr }
			return d.DeleteScheduledQuery(ctx, 1)
		},
	},
	{
		name:           "DeleteLabel",
		want:           "scope global labels",
		alwaysNotifies: true,
		run: func(ctx context.Context, ds *mock.Store, d *Datastore, unchanged bool, writeErr error) error {
			ds.DeleteLabelFunc = func(ctx context.Context, name string, filter fleet.TeamFilter) error { return writeErr }
			return d.DeleteLabel(ctx, "l", fleet.TeamFilter{})
		},
	},
}

func TestHooks(t *testing.T) {
	for _, c := range hookCases {
		t.Run(c.name, func(t *testing.T) {
			t.Run("changed", func(t *testing.T) {
				ds, notifier, d := newTestDatastore()
				require.NoError(t, c.run(t.Context(), ds, d, false, nil))
				require.Equal(t, []string{c.want}, notifier.calls)
			})

			t.Run("unchanged", func(t *testing.T) {
				ds, notifier, d := newTestDatastore()
				require.NoError(t, c.run(t.Context(), ds, d, true, nil))
				if c.alwaysNotifies {
					require.Equal(t, []string{c.want}, notifier.calls)
				} else {
					require.Empty(t, notifier.calls)
				}
			})

			t.Run("write fails", func(t *testing.T) {
				ds, notifier, d := newTestDatastore()
				require.Error(t, c.run(t.Context(), ds, d, false, errors.New("boom")))
				if c.notifiesOnError {
					require.Equal(t, []string{c.want}, notifier.calls)
				} else {
					require.Empty(t, notifier.calls)
				}
			})
		})
	}
}

func TestSaveAppConfigQueryReportsDisabled(t *testing.T) {
	ds, notifier, d := newTestDatastore()
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return &fleet.AppConfig{}, nil }
	ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error { return nil }
	after := &fleet.AppConfig{}
	after.ServerSettings.QueryReportsDisabled = true
	require.NoError(t, d.SaveAppConfig(t.Context(), after))
	require.Equal(t, []string{"scope global settings"}, notifier.calls)
}

func TestUnreadableBeforeNotifies(t *testing.T) {
	t.Run("SaveAppConfig", func(t *testing.T) {
		ds, notifier, d := newTestDatastore()
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return nil, errors.New("boom") }
		ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error { return nil }
		require.NoError(t, d.SaveAppConfig(t.Context(), &fleet.AppConfig{}))
		require.Equal(t, []string{"scope global settings"}, notifier.calls)
	})

	t.Run("reports digest", func(t *testing.T) {
		ds, notifier, d := newTestDatastore()
		ds.ScheduledQueriesDigestFunc = func(ctx context.Context, teamID *uint) (string, error) {
			return "", errors.New("boom")
		}
		ds.SaveQueryFunc = func(ctx context.Context, query *fleet.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
			return nil
		}
		require.NoError(t, d.SaveQuery(t.Context(), &fleet.Query{Saved: true, Interval: 60}, false, false))
		require.Equal(t, []string{"scope global reports"}, notifier.calls)
	})
}

func TestUnscheduledReportsSkipDigest(t *testing.T) {
	ds, notifier, d := newTestDatastore()
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		return query, nil
	}
	ds.SaveQueryFunc = func(ctx context.Context, query *fleet.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
		return nil
	}
	var stored fleet.Query
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) { return &stored, nil }
	write := mockReportsDigest(ds, false)

	_, err := d.NewQuery(t.Context(), &fleet.Query{Saved: true})
	require.NoError(t, err)
	require.NoError(t, d.SaveQuery(t.Context(), &fleet.Query{Saved: true}, false, false))
	require.False(t, ds.ScheduledQueriesDigestFuncInvoked)
	require.Empty(t, notifier.calls)

	// Unscheduling a scheduled report changes the schedule.
	stored = fleet.Query{Saved: true, Interval: 60}
	ds.SaveQueryFunc = func(ctx context.Context, query *fleet.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
		write()
		return nil
	}
	require.NoError(t, d.SaveQuery(t.Context(), &fleet.Query{Saved: true}, false, false))
	require.Equal(t, []string{"scope global reports"}, notifier.calls)
}

func TestRawChanged(t *testing.T) {
	raw := func(s string) *json.RawMessage { return new(json.RawMessage(s)) }
	require.False(t, rawChanged(nil, nil))
	require.False(t, rawChanged(raw(`{"a":1,"b":{"c":2,"d":3}}`), raw(`{ "b": {"d":3,"c":2}, "a":1 }`)))
	require.True(t, rawChanged(raw(`{"a":1}`), raw(`{"a":2}`)))
	require.True(t, rawChanged(raw(`{"a":12345678901234567890}`), raw(`{"a":12345678901234567891}`)))
	require.True(t, rawChanged(nil, raw(`{}`)))
	require.True(t, rawChanged(raw(`{`), raw(`{`)))
}

func TestApplyQueriesNotifiesChangedScopes(t *testing.T) {
	ds, notifier, d := newTestDatastore()
	written := false
	ds.ScheduledQueriesDigestFunc = func(ctx context.Context, teamID *uint) (string, error) {
		// Only fleet 3's reports change.
		if written && teamID != nil && *teamID == 3 {
			return "after", nil
		}
		return "before", nil
	}
	ds.ApplyQueriesFunc = func(ctx context.Context, authorID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]struct{}) error {
		written = true
		return nil
	}
	require.NoError(t, d.ApplyQueries(t.Context(), 1, []*fleet.Query{
		{Name: "a"},
		{Name: "b", TeamID: new(uint(3))},
		{Name: "c", TeamID: new(uint(4))},
		{Name: "d", TeamID: new(uint(3))},
	}, nil))
	require.Equal(t, []string{"scope team:3 reports"}, notifier.calls)
}

// TestEveryWrappedMethodHasAHookCase requires each method declared by the
// decorator to have a hook case proving it notifies (see the etag_invalidate
// test of the same name for why unwrapped methods can't be detected).
func TestEveryWrappedMethodHasAHookCase(t *testing.T) {
	covered := map[string]struct{}{}
	for _, c := range hookCases {
		covered[c.name] = struct{}{}
	}

	declared := declaredMethods(t, ".")
	require.NotEmpty(t, declared, "found no declared decorator methods — did the receiver name change?")
	var missing []string
	for _, m := range declared {
		if _, ok := covered[m]; !ok {
			missing = append(missing, m)
		}
	}
	sort.Strings(missing)
	require.Empty(t, missing, "add a hook case for these wrapped methods: %v", missing)
}

// TestCoversETagInvalidation requires every write that invalidates osquery
// config ETags (i.e. that can change the osquery config) to notify agents
// too, or to be explicitly skipped.
func TestCoversETagInvalidation(t *testing.T) {
	skipped := map[string]string{
		"NewTeam": "a new fleet has no hosts",
		// Label membership changes: fallback poll only.
		"ApplyLabelSpecs":                     "label membership",
		"ApplyLabelSpecsWithAuthor":           "label membership",
		"SaveLabel":                           "label membership",
		"UpdateLabelMembershipByHostIDs":      "label membership",
		"UpdateLabelMembershipByHostCriteria": "label membership",
		"AddLabelsToHost":                     "label membership",
		"RemoveLabelsFromHost":                "label membership",
		"RecordLabelQueryExecutions":          "label membership",
		"AsyncBatchInsertLabelMembership":     "label membership",
		"AsyncBatchDeleteLabelMembership":     "label membership",
	}
	ours := map[string]struct{}{}
	for _, m := range declaredMethods(t, ".") {
		ours[m] = struct{}{}
	}
	etagMethods := declaredMethods(t, filepath.Join("..", "etag_invalidate"))
	require.NotEmpty(t, etagMethods)
	var missing []string
	for _, m := range etagMethods {
		_, wrapped := ours[m]
		_, skip := skipped[m]
		if !wrapped && !skip {
			missing = append(missing, m)
		}
	}
	sort.Strings(missing)
	require.Empty(t, missing, "wrap these etag_invalidate methods or add them to the skipped list: %v", missing)
}

// declaredMethods returns the exported methods declared on *Datastore by the
// non-test Go files of dir.
func declaredMethods(t *testing.T, dir string) []string {
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	declRE := regexp.MustCompile(`(?m)^func \(d \*Datastore\) ([A-Z]\w*)\(`)
	var declared []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		for _, m := range declRE.FindAllStringSubmatch(string(src), -1) {
			declared = append(declared, m[1])
		}
	}
	return declared
}
