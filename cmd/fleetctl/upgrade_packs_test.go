package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestUpgradeSinglePack(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		desc      string
		pack      *fleet.Pack
		queries   []*fleet.Query
		scheds    []fleet.PackSpecQuery
		want      []*fleet.QuerySpec
		wantCount int
	}{
		{
			desc:      "no queries, a target",
			pack:      &fleet.Pack{Name: "p1", Teams: []fleet.Target{{Type: fleet.TargetTeam, DisplayText: "t1"}}},
			queries:   nil,
			scheds:    nil,
			want:      nil,
			wantCount: 0,
		},
		{
			desc:      "no queries, no target",
			pack:      &fleet.Pack{Name: "p1"},
			queries:   nil,
			scheds:    nil,
			want:      nil,
			wantCount: 0,
		},
		{
			desc:      "a query, no target",
			pack:      &fleet.Pack{Name: "p1"},
			queries:   []*fleet.Query{{Name: "q1", Query: "select 1"}},
			scheds:    []fleet.PackSpecQuery{{QueryName: "q1", Interval: 60}},
			want:      nil,
			wantCount: 0,
		},
		{
			desc:    "a query, label target",
			pack:    &fleet.Pack{Name: "p1", Labels: []fleet.Target{{Type: fleet.TargetLabel, DisplayText: "l1"}}},
			queries: []*fleet.Query{{Name: "q1", Query: "select 1"}},
			scheds:  []fleet.PackSpecQuery{{QueryName: "q1", Interval: 60}},
			want: []*fleet.QuerySpec{
				// global query, schedule is removed
				{Name: "p1 - q1 - Jan  1 00:00:00.000", Description: `(converted from pack "p1", query "q1")`, Query: "select 1", Interval: 0},
			},
			wantCount: 1,
		},
		{
			desc: "2 queries, host target",
			pack: &fleet.Pack{Name: "p1", Hosts: []fleet.Target{{Type: fleet.TargetHost, DisplayText: "h1"}}},
			queries: []*fleet.Query{
				{Name: "q1", Query: "select 1"},
				{Name: "q2", Query: "select 2", ObserverCanRun: true, Description: "q2 desc"},
			},
			scheds: []fleet.PackSpecQuery{
				{QueryName: "q1", Interval: 60, Name: "sq1", Snapshot: ptr.Bool(true), Platform: ptr.String("darwin"), Version: ptr.String("v1")},
				{QueryName: "q2", Interval: 90, Name: "sq2", Description: "sq2 desc"},
			},
			want: []*fleet.QuerySpec{
				// global queries, schedule is removed
				{Name: "p1 - q1 - Jan  1 00:00:00.000", Description: `(converted from pack "p1", query "q1")`, Query: "select 1", Interval: 0, Logging: "snapshot", Platform: "darwin", MinOsqueryVersion: "v1"},
				{Name: "p1 - q2 - Jan  1 00:00:00.000", Description: "q2 desc\n(converted from pack \"p1\", query \"q2\")", Query: "select 2", Interval: 0, ObserverCanRun: true},
			},
			wantCount: 2,
		},
		{
			desc: "2 queries, 2 team targets",
			pack: &fleet.Pack{Name: "p1", Description: "p1 desc", Platform: "ignored", Teams: []fleet.Target{
				{Type: fleet.TargetTeam, DisplayText: "t1"},
				{Type: fleet.TargetTeam, DisplayText: "t2"},
			}},
			queries: []*fleet.Query{
				{Name: "q1", Query: "select 1"},
				{Name: "q2", Query: "select 2", ObserverCanRun: true, Description: "q2 desc"},
			},
			scheds: []fleet.PackSpecQuery{
				{QueryName: "q1", Interval: 60, Name: "sq1", Snapshot: ptr.Bool(true), Removed: ptr.Bool(true), Platform: ptr.String("darwin"), Version: ptr.String("v1")},
				{QueryName: "q2", Interval: 90, Name: "sq2", Removed: ptr.Bool(false), Description: "sq2 desc"},
			},
			want: []*fleet.QuerySpec{
				// per-team queries
				{Name: "p1 - q1 - t1 - Jan  1 00:00:00.000", Description: `(converted from pack "p1", query "q1")`, TeamName: "t1", AutomationsEnabled: true, Query: "select 1", Interval: 60, Logging: "snapshot", Platform: "darwin", MinOsqueryVersion: "v1"},
				{Name: "p1 - q1 - t2 - Jan  1 00:00:00.000", Description: `(converted from pack "p1", query "q1")`, TeamName: "t2", AutomationsEnabled: true, Query: "select 1", Interval: 60, Logging: "snapshot", Platform: "darwin", MinOsqueryVersion: "v1"},
				{Name: "p1 - q2 - t1 - Jan  1 00:00:00.000", Description: "q2 desc\n(converted from pack \"p1\", query \"q2\")", TeamName: "t1", AutomationsEnabled: true, Query: "select 2", Interval: 90, ObserverCanRun: true, Logging: "differential_ignore_removals"},
				{Name: "p1 - q2 - t2 - Jan  1 00:00:00.000", Description: "q2 desc\n(converted from pack \"p1\", query \"q2\")", TeamName: "t2", AutomationsEnabled: true, Query: "select 2", Interval: 90, ObserverCanRun: true, Logging: "differential_ignore_removals"},
			},
			wantCount: 2,
		},
		{
			desc: "2 queries, 2 team targets, label target",
			pack: &fleet.Pack{Name: "p1", Description: "p1 desc", Platform: "ignored", Teams: []fleet.Target{
				{Type: fleet.TargetTeam, DisplayText: "t1"},
				{Type: fleet.TargetTeam, DisplayText: "t2"},
			}, Labels: []fleet.Target{
				{Type: fleet.TargetLabel, DisplayText: "l1"},
			}},
			queries: []*fleet.Query{
				{Name: "q1", Query: "select 1"},
				{Name: "q2", Query: "select 2", ObserverCanRun: true, Description: "q2 desc"},
			},
			scheds: []fleet.PackSpecQuery{
				{QueryName: "q1", Interval: 60, Name: "sq1", Snapshot: ptr.Bool(true), Removed: ptr.Bool(true), Platform: ptr.String("darwin"), Version: ptr.String("v1")},
				{QueryName: "q2", Interval: 90, Name: "sq2", Removed: ptr.Bool(false), Description: "sq2 desc"},
			},
			want: []*fleet.QuerySpec{
				// per-team queries, and global queries with schedules removed
				{Name: "p1 - q1 - t1 - Jan  1 00:00:00.000", Description: `(converted from pack "p1", query "q1")`, TeamName: "t1", AutomationsEnabled: true, Query: "select 1", Interval: 60, Logging: "snapshot", Platform: "darwin", MinOsqueryVersion: "v1"},
				{Name: "p1 - q1 - t2 - Jan  1 00:00:00.000", Description: `(converted from pack "p1", query "q1")`, TeamName: "t2", AutomationsEnabled: true, Query: "select 1", Interval: 60, Logging: "snapshot", Platform: "darwin", MinOsqueryVersion: "v1"},
				{Name: "p1 - q1 - Jan  1 00:00:00.000", Description: `(converted from pack "p1", query "q1")`, Query: "select 1", Interval: 0, Logging: "snapshot", Platform: "darwin", MinOsqueryVersion: "v1"},
				{Name: "p1 - q2 - t1 - Jan  1 00:00:00.000", Description: "q2 desc\n(converted from pack \"p1\", query \"q2\")", TeamName: "t1", AutomationsEnabled: true, Query: "select 2", Interval: 90, ObserverCanRun: true, Logging: "differential_ignore_removals"},
				{Name: "p1 - q2 - t2 - Jan  1 00:00:00.000", Description: "q2 desc\n(converted from pack \"p1\", query \"q2\")", TeamName: "t2", AutomationsEnabled: true, Query: "select 2", Interval: 90, ObserverCanRun: true, Logging: "differential_ignore_removals"},
				{Name: "p1 - q2 - Jan  1 00:00:00.000", Description: "q2 desc\n(converted from pack \"p1\", query \"q2\")", Query: "select 2", Interval: 0, ObserverCanRun: true, Logging: "differential_ignore_removals"},
			},
			wantCount: 2,
		},
		{
			desc: "2 queries, team target, host target",
			pack: &fleet.Pack{Name: "p1", Description: "p1 desc", Platform: "ignored", Teams: []fleet.Target{
				{Type: fleet.TargetTeam, DisplayText: "t1"},
			}, Hosts: []fleet.Target{
				{Type: fleet.TargetHost, DisplayText: "h1"},
			}},
			queries: []*fleet.Query{
				{Name: "q1", Query: "select 1"},
				{Name: "q2", Query: "select 2", ObserverCanRun: true, Description: "q2 desc"},
			},
			scheds: []fleet.PackSpecQuery{
				{QueryName: "q1", Interval: 60, Name: "sq1", Removed: ptr.Bool(true), Platform: ptr.String("darwin"), Version: ptr.String("v1")},
				{QueryName: "q2", Interval: 90, Name: "sq2", Removed: ptr.Bool(false), Description: "sq2 desc"},
			},
			want: []*fleet.QuerySpec{
				// per-team queries, and global queries with schedules removed
				{Name: "p1 - q1 - t1 - Jan  1 00:00:00.000", Description: `(converted from pack "p1", query "q1")`, TeamName: "t1", AutomationsEnabled: true, Query: "select 1", Interval: 60, Logging: "differential", Platform: "darwin", MinOsqueryVersion: "v1"},
				{Name: "p1 - q1 - Jan  1 00:00:00.000", Description: `(converted from pack "p1", query "q1")`, Query: "select 1", Interval: 0, Logging: "differential", Platform: "darwin", MinOsqueryVersion: "v1"},
				{Name: "p1 - q2 - t1 - Jan  1 00:00:00.000", Description: "q2 desc\n(converted from pack \"p1\", query \"q2\")", TeamName: "t1", AutomationsEnabled: true, Query: "select 2", Interval: 90, ObserverCanRun: true, Logging: "differential_ignore_removals"},
				{Name: "p1 - q2 - Jan  1 00:00:00.000", Description: "q2 desc\n(converted from pack \"p1\", query \"q2\")", Query: "select 2", Interval: 0, ObserverCanRun: true, Logging: "differential_ignore_removals"},
			},
			wantCount: 2,
		},
		{
			desc: "2 queries, all targets, a query with no schedule match",
			pack: &fleet.Pack{Name: "p1", Description: "p1 desc", Platform: "ignored", Teams: []fleet.Target{
				{Type: fleet.TargetTeam, DisplayText: "t1"},
			}, Hosts: []fleet.Target{
				{Type: fleet.TargetHost, DisplayText: "h1"},
			}, Labels: []fleet.Target{
				{Type: fleet.TargetLabel, DisplayText: "l1"},
			}},
			queries: []*fleet.Query{
				{Name: "q1", Query: "select 1"},
				{Name: "q2", Query: "select 2", ObserverCanRun: true, Description: "q2 desc"},
			},
			scheds: []fleet.PackSpecQuery{
				{QueryName: "q1", Interval: 60, Name: "sq1", Removed: ptr.Bool(true), Platform: ptr.String("darwin"), Version: ptr.String("v1")},
				{QueryName: "no-such-query", Interval: 90, Name: "sq2", Removed: ptr.Bool(false), Description: "sq2 desc"},
			},
			want: []*fleet.QuerySpec{
				// per-team queries, and global queries with schedules removed
				{Name: "p1 - q1 - t1 - Jan  1 00:00:00.000", Description: `(converted from pack "p1", query "q1")`, TeamName: "t1", AutomationsEnabled: true, Query: "select 1", Interval: 60, Logging: "differential", Platform: "darwin", MinOsqueryVersion: "v1"},
				{Name: "p1 - q1 - Jan  1 00:00:00.000", Description: `(converted from pack "p1", query "q1")`, Query: "select 1", Interval: 0, Logging: "differential", Platform: "darwin", MinOsqueryVersion: "v1"},
			},
			wantCount: 1,
		},
		{
			desc:    "a query, team target, disabled pack",
			pack:    &fleet.Pack{Name: "p1", Disabled: true, Teams: []fleet.Target{{Type: fleet.TargetTeam, DisplayText: "t1"}}},
			queries: []*fleet.Query{{Name: "q1", Query: "select 1"}},
			scheds:  []fleet.PackSpecQuery{{QueryName: "q1", Interval: 60}},
			want: []*fleet.QuerySpec{
				{Name: "p1 - q1 - t1 - Jan  1 00:00:00.000", Description: `(converted from pack "p1", query "q1")`, Query: "select 1", TeamName: "t1", AutomationsEnabled: false, Interval: 60},
			},
			wantCount: 1,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			// create the pack spec corresponding to the DB pack of the case
			packSpec := &fleet.PackSpec{
				Name:        c.pack.Name,
				Description: c.pack.Description,
				Platform:    c.pack.Platform,
				Disabled:    c.pack.Disabled,
				Queries:     c.scheds,
			}
			for _, tt := range c.pack.Teams {
				packSpec.Targets.Teams = append(packSpec.Targets.Teams, tt.DisplayText)
			}
			for _, lt := range c.pack.Labels {
				packSpec.Targets.Labels = append(packSpec.Targets.Labels, lt.DisplayText)
			}

			got, n := upgradePackToQueriesSpecs(packSpec, c.pack, c.queries, ts)

			// Equal gives a better diff than ElementsMatch, so for maintainability of the
			// test, it's worth it to keep the expected results in the same order as the
			// actual ones.
			require.Equal(t, c.want, got)
			require.Equal(t, c.wantCount, n)
		})
	}
}

func TestFleetctlUpgradePacks_EmptyPacks(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{ServerURL: "https://example.com"}}, nil
	}

	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return &fleet.User{ID: id, GlobalRole: ptr.String(fleet.RoleAdmin)}, nil
	}

	ds.GetPackSpecsFunc = func(ctx context.Context) ([]*fleet.PackSpec, error) {
		return []*fleet.PackSpec{
			{Name: "p1", Targets: fleet.PackSpecTargets{Labels: []string{"l1"}}},
			{Name: "p2", Targets: fleet.PackSpecTargets{Teams: []string{"t1"}}},
		}, nil
	}

	ds.ListPacksFunc = func(ctx context.Context, opt fleet.PackListOptions) ([]*fleet.Pack, error) {
		return []*fleet.Pack{
			{Name: "p1", Labels: []fleet.Target{{Type: fleet.TargetLabel, DisplayText: "l1"}}, LabelIDs: []uint{1}},
			{Name: "p2", Teams: []fleet.Target{{Type: fleet.TargetTeam, DisplayText: "t1"}}, TeamIDs: []uint{1}},
		}, nil
	}

	ds.ListScheduledQueriesInPackWithStatsFunc = func(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
		return nil, nil
	}

	ds.CountHostsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{}, nil
	}

	ds.ListQueriesFunc = func(ctx context.Context, opt fleet.ListQueryOptions) ([]*fleet.Query, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.yml")

	// write some dummy data in the file, it should be overwritten
	err := os.WriteFile(outputFile, []byte("dummy"), 0o644)
	require.NoError(t, err)

	got := runAppForTest(t, []string{"upgrade-packs", "-o", outputFile})
	require.Contains(t, got, `Converted 0 queries from 2 2017 "Packs" into portable queries:`)
	require.Contains(t, got, `visit https://example.com/packs/manage and disable all`)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	require.Empty(t, content)
}

func TestFleetctlUpgradePacks_NonEmpty(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return &fleet.User{ID: id, GlobalRole: ptr.String(fleet.RoleAdmin)}, nil
	}

	ds.GetPackSpecsFunc = func(ctx context.Context) ([]*fleet.PackSpec, error) {
		// queries must match those returned by ListScheduledQueriesInPackWithStats
		return []*fleet.PackSpec{
			{ID: 1, Name: "p1", Targets: fleet.PackSpecTargets{Labels: []string{"l1"}}, Queries: []fleet.PackSpecQuery{
				{QueryName: "q1", Name: "sq1", Interval: 60, Snapshot: ptr.Bool(true), Platform: ptr.String("darwin")},
			}},
			{ID: 2, Name: "p2", Targets: fleet.PackSpecTargets{Teams: []string{"t1", "t2"}}, Queries: []fleet.PackSpecQuery{
				{QueryName: "q2", Name: "sq2", Interval: 90, Removed: ptr.Bool(true), Platform: ptr.String("linux")},
			}},
		}, nil
	}

	ds.ListPacksFunc = func(ctx context.Context, opt fleet.PackListOptions) ([]*fleet.Pack, error) {
		return []*fleet.Pack{
			{ID: 1, Name: "p1", Labels: []fleet.Target{
				{Type: fleet.TargetLabel, DisplayText: "l1"},
			}, LabelIDs: []uint{1}},
			{ID: 2, Name: "p2", Teams: []fleet.Target{
				{Type: fleet.TargetTeam, DisplayText: "t1"},
				{Type: fleet.TargetTeam, DisplayText: "t2"},
			}, TeamIDs: []uint{1, 2}},
		}, nil
	}

	ds.ListScheduledQueriesInPackWithStatsFunc = func(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
		// queries must match those returned by GetPackSpecs
		if id == 1 {
			return []*fleet.ScheduledQuery{
				{ID: 1, PackID: id, Name: "sq1", QueryName: "q1", Interval: 60, Snapshot: ptr.Bool(true), Platform: ptr.String("darwin")},
			}, nil
		}
		return []*fleet.ScheduledQuery{
			{ID: 2, PackID: id, Name: "sq2", QueryName: "q2", Interval: 90, Removed: ptr.Bool(true), Platform: ptr.String("linux")},
		}, nil
	}

	ds.CountHostsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{}, nil
	}

	ds.ListQueriesFunc = func(ctx context.Context, opt fleet.ListQueryOptions) ([]*fleet.Query, int, *fleet.PaginationMetadata, error) {
		return []*fleet.Query{
			{Name: "q1", Query: "select 1"},
			{Name: "q2", Query: "select 2"},
			{Name: "q3", Query: "select 3"},
		}, 3, nil, nil
	}

	const expected = `
apiVersion: v1
kind: query
spec:
  automations_enabled: false
  description: (converted from pack "p1", query "q1")
  discard_data: false
  interval: 0
  logging: snapshot
  min_osquery_version: ""
  name: p1 - q1 - Jan  1 00:00:00.000
  observer_can_run: false
  platform: darwin
  query: select 1
  team: ""
---
apiVersion: v1
kind: query
spec:
  automations_enabled: true
  description: (converted from pack "p2", query "q2")
  discard_data: false
  interval: 90
  logging: differential
  min_osquery_version: ""
  name: p2 - q2 - t1 - Jan  1 00:00:00.000
  observer_can_run: false
  platform: linux
  query: select 2
  team: t1
---
apiVersion: v1
kind: query
spec:
  automations_enabled: true
  description: (converted from pack "p2", query "q2")
  discard_data: false
  interval: 90
  logging: differential
  min_osquery_version: ""
  name: p2 - q2 - t2 - Jan  1 00:00:00.000
  observer_can_run: false
  platform: linux
  query: select 2
  team: t2
---
`

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.yml")

	// write some dummy data in the file, it should be overwritten
	err := os.WriteFile(outputFile, []byte("dummy"), 0o644)
	require.NoError(t, err)

	testUpgradePacksTimestamp = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	got := runAppForTest(t, []string{"upgrade-packs", "-o", outputFile})
	require.Contains(t, got, `Converted 2 queries from 2 2017 "Packs" into portable queries:`)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(content)))
}

func TestFleetctlUpgradePacks_NotAdmin(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return &fleet.User{ID: id, GlobalRole: ptr.String(fleet.RoleObserver)}, nil
	}

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.yml")

	// write some dummy data in the file, it should NOT be overwritten
	err := os.WriteFile(outputFile, []byte("dummy"), 0o644)
	require.NoError(t, err)

	// first try without the required output file flag
	runAppCheckErr(t, []string{"upgrade-packs"}, `Required flag "o" not set`)
	// then try with the required flag but user is not admin
	runAppCheckErr(t, []string{"upgrade-packs", "-o", outputFile}, `could not upgrade packs: forbidden: user does not have the admin role`)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	require.Equal(t, []byte("dummy"), content)
}

func TestFleetctlUpgradePacks_NoPack(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return &fleet.User{ID: id, GlobalRole: ptr.String(fleet.RoleAdmin)}, nil
	}

	ds.GetPackSpecsFunc = func(ctx context.Context) ([]*fleet.PackSpec, error) {
		return nil, nil
	}

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.yml")

	// write some dummy data in the file, it should NOT be overwritten
	err := os.WriteFile(outputFile, []byte("dummy"), 0o644)
	require.NoError(t, err)

	got := runAppForTest(t, []string{"upgrade-packs", "-o", outputFile})
	require.Contains(t, got, "No 2017 \"Packs\" found.\n")

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	require.Equal(t, []byte("dummy"), content)
}
