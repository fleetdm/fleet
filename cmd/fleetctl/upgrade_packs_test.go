package main

import (
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

func TestFleetctlUpgradePacks_Empty(t *testing.T) {
}

func TestFleetctlUpgradePacks_NonEmpty(t *testing.T) {
}
