package etag_invalidate

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

type stubStore struct {
	fleet.ConfigETagStore
	invalidateCalls       int
	legacyResetCalls      int
	labelScopesResetCalls int
	hostInvalidations     []uint
	invalidateErr         error
	resetErr              error
	hostErr               error
}

func (s *stubStore) Invalidate(ctx context.Context) error {
	s.invalidateCalls++
	return s.invalidateErr
}

func (s *stubStore) ResetLegacyPacksFlag(ctx context.Context) error {
	s.legacyResetCalls++
	return s.resetErr
}

func (s *stubStore) ResetLabelScopes(ctx context.Context) error {
	s.labelScopesResetCalls++
	return s.resetErr
}

func (s *stubStore) InvalidateHost(ctx context.Context, hostID uint) error {
	s.hostInvalidations = append(s.hostInvalidations, hostID)
	return s.hostErr
}

func newTestDatastore() (*mock.Store, *stubStore, *Datastore) {
	ds := new(mock.Store)
	store := &stubStore{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return ds, store, New(ds, store, logger)
}

// TestDeploymentHooksFireOnSuccess exercises every deployment-wide hook: the
// write must delegate to the wrapped datastore and fire a deployment
// invalidation on success. Query methods (and DeleteLabel) must additionally
// reset the label-scope mode state; pack methods must reset the legacy-packs
// gate flag.
//
// If you add a method to the etag_invalidate decorator, add a case here.

// hookTestCtx is the context the hook case closures run under. The tables live
// at package level so TestEveryWrappedMethodHasAHookCase can read them: marking
// a method as wrapped and proving it invalidates are then the same act.
var hookTestCtx = context.Background()

const (
	resetsNothing = iota
	resetsLegacyFlag
	resetsLabelScopes
)

type deploymentHookCase struct {
	name        string
	resets      int
	setupAndRun func(ds *mock.Store, d *Datastore) (invoked *bool, run func() error)
}

var deploymentHookCases = []deploymentHookCase{
	{
		"SaveAppConfig", resetsNothing,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error { return nil }
			return &ds.SaveAppConfigFuncInvoked, func() error { return d.SaveAppConfig(hookTestCtx, &fleet.AppConfig{}) }
		},
	},
	{
		"NewTeam", resetsNothing,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) { return team, nil }
			return &ds.NewTeamFuncInvoked, func() error { _, err := d.NewTeam(hookTestCtx, &fleet.Team{}); return err }
		},
	},
	{
		"SaveTeam", resetsNothing,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) { return team, nil }
			return &ds.SaveTeamFuncInvoked, func() error { _, err := d.SaveTeam(hookTestCtx, &fleet.Team{}); return err }
		},
	},
	{
		"DeleteTeam", resetsNothing,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.DeleteTeamFunc = func(ctx context.Context, tid uint) error { return nil }
			return &ds.DeleteTeamFuncInvoked, func() error { return d.DeleteTeam(hookTestCtx, 1) }
		},
	},
	{
		"ApplyQueries", resetsLabelScopes,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.ApplyQueriesFunc = func(ctx context.Context, authorID uint, queries []*fleet.Query, discard map[uint]struct{}) error {
				return nil
			}
			return &ds.ApplyQueriesFuncInvoked, func() error { return d.ApplyQueries(hookTestCtx, 1, nil, nil) }
		},
	},
	{
		"NewQuery", resetsLabelScopes,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
				return query, nil
			}
			return &ds.NewQueryFuncInvoked, func() error { _, err := d.NewQuery(hookTestCtx, &fleet.Query{}); return err }
		},
	},
	{
		"SaveQuery", resetsLabelScopes,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.SaveQueryFunc = func(ctx context.Context, query *fleet.Query, discard, deleteStats bool) error { return nil }
			return &ds.SaveQueryFuncInvoked, func() error { return d.SaveQuery(hookTestCtx, &fleet.Query{}, false, false) }
		},
	},
	{
		"DeleteQuery", resetsLabelScopes,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.DeleteQueryFunc = func(ctx context.Context, teamID *uint, name string) error { return nil }
			return &ds.DeleteQueryFuncInvoked, func() error { return d.DeleteQuery(hookTestCtx, nil, "q") }
		},
	},
	{
		"DeleteQueries", resetsLabelScopes,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.DeleteQueriesFunc = func(ctx context.Context, ids []uint) (uint, error) { return 1, nil }
			return &ds.DeleteQueriesFuncInvoked, func() error { _, err := d.DeleteQueries(hookTestCtx, []uint{1}); return err }
		},
	},
	{
		"ApplyPackSpecs", resetsLegacyFlag,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.ApplyPackSpecsFunc = func(ctx context.Context, specs []*fleet.PackSpec) error { return nil }
			return &ds.ApplyPackSpecsFuncInvoked, func() error { return d.ApplyPackSpecs(hookTestCtx, nil) }
		},
	},
	{
		"NewPack", resetsLegacyFlag,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.NewPackFunc = func(ctx context.Context, pack *fleet.Pack, opts ...fleet.OptionalArg) (*fleet.Pack, error) {
				return pack, nil
			}
			return &ds.NewPackFuncInvoked, func() error { _, err := d.NewPack(hookTestCtx, &fleet.Pack{}); return err }
		},
	},
	{
		"SavePack", resetsLegacyFlag,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.SavePackFunc = func(ctx context.Context, pack *fleet.Pack) error { return nil }
			return &ds.SavePackFuncInvoked, func() error { return d.SavePack(hookTestCtx, &fleet.Pack{}) }
		},
	},
	{
		"DeletePack", resetsLegacyFlag,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.DeletePackFunc = func(ctx context.Context, name string) error { return nil }
			return &ds.DeletePackFuncInvoked, func() error { return d.DeletePack(hookTestCtx, "p") }
		},
	},
	{
		"NewScheduledQuery", resetsNothing,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.NewScheduledQueryFunc = func(ctx context.Context, sq *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
				return sq, nil
			}
			return &ds.NewScheduledQueryFuncInvoked, func() error { _, err := d.NewScheduledQuery(hookTestCtx, &fleet.ScheduledQuery{}); return err }
		},
	},
	{
		"SaveScheduledQuery", resetsNothing,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.SaveScheduledQueryFunc = func(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) { return sq, nil }
			return &ds.SaveScheduledQueryFuncInvoked, func() error { _, err := d.SaveScheduledQuery(hookTestCtx, &fleet.ScheduledQuery{}); return err }
		},
	},
	{
		"DeleteScheduledQuery", resetsNothing,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.DeleteScheduledQueryFunc = func(ctx context.Context, id uint) error { return nil }
			return &ds.DeleteScheduledQueryFuncInvoked, func() error { return d.DeleteScheduledQuery(hookTestCtx, 1) }
		},
	},
	{
		// The GitOps label path can change membership immediately
		// (platform changes delete it; manual-label specs replace it)
		// with removals unknowable from the specs — deployment-wide.
		"ApplyLabelSpecs", resetsLabelScopes,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.ApplyLabelSpecsFunc = func(ctx context.Context, specs []*fleet.LabelSpec) error { return nil }
			return &ds.ApplyLabelSpecsFuncInvoked, func() error {
				return d.ApplyLabelSpecs(hookTestCtx, []*fleet.LabelSpec{{Name: "l"}})
			}
		},
	},
	{
		"ApplyLabelSpecsWithAuthor", resetsLabelScopes,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.ApplyLabelSpecsWithAuthorFunc = func(ctx context.Context, specs []*fleet.LabelSpec, authorID *uint) error { return nil }
			return &ds.ApplyLabelSpecsWithAuthorFuncInvoked, func() error {
				return d.ApplyLabelSpecsWithAuthor(hookTestCtx, []*fleet.LabelSpec{{Name: "l"}}, new(uint(1)))
			}
		},
	},
	{
		// DeleteLabel is a REQUIRED hook: deleting a referenced label
		// cascades query_labels away and can instantly unscope a report.
		"DeleteLabel", resetsLabelScopes,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.DeleteLabelFunc = func(ctx context.Context, name string, filter fleet.TeamFilter) error { return nil }
			return &ds.DeleteLabelFuncInvoked, func() error { return d.DeleteLabel(hookTestCtx, "l", fleet.TeamFilter{}) }
		},
	},
	{
		// SaveLabel uses the deployment-wide invalidation: the supplied
		// host list contains only NEW membership, so removed hosts would
		// be missed by per-host invalidation.
		"SaveLabel", resetsNothing,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.SaveLabelFunc = func(ctx context.Context, label *fleet.Label, hostIDs []uint, tf fleet.TeamFilter) (*fleet.LabelWithTeamName, []uint, error) {
				return &fleet.LabelWithTeamName{}, nil, nil
			}
			return &ds.SaveLabelFuncInvoked, func() error {
				_, _, err := d.SaveLabel(hookTestCtx, &fleet.Label{}, []uint{1, 2}, fleet.TeamFilter{})
				return err
			}
		},
	},
	{
		"UpdateLabelMembershipByHostIDs", resetsNothing,
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.UpdateLabelMembershipByHostIDsFunc = func(ctx context.Context, label fleet.Label, hostIds []uint, tf fleet.TeamFilter) (*fleet.Label, []uint, error) {
				return &fleet.Label{}, nil, nil
			}
			return &ds.UpdateLabelMembershipByHostIDsFuncInvoked, func() error {
				_, _, err := d.UpdateLabelMembershipByHostIDs(hookTestCtx, fleet.Label{}, []uint{1}, fleet.TeamFilter{})
				return err
			}
		},
	},
}

type perHostHookCase struct {
	name        string
	wantHosts   []uint
	setupAndRun func(ds *mock.Store, d *Datastore) (invoked *bool, run func() error)
}

var perHostHookCases = []perHostHookCase{
	{
		"RecordLabelQueryExecutions", []uint{11},
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.RecordLabelQueryExecutionsFunc = func(ctx context.Context, host *fleet.Host, results map[uint]*bool, t time.Time, deferred bool) error {
				return nil
			}
			return &ds.RecordLabelQueryExecutionsFuncInvoked, func() error {
				return d.RecordLabelQueryExecutions(hookTestCtx, &fleet.Host{ID: 11}, nil, time.Now(), false)
			}
		},
	},
	{
		// batch tuples are [labelID, hostID]; distinct hosts only
		"AsyncBatchInsertLabelMembership", []uint{21, 22},
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.AsyncBatchInsertLabelMembershipFunc = func(ctx context.Context, batch [][2]uint) error { return nil }
			return &ds.AsyncBatchInsertLabelMembershipFuncInvoked, func() error {
				return d.AsyncBatchInsertLabelMembership(hookTestCtx, [][2]uint{{1, 21}, {2, 21}, {1, 22}})
			}
		},
	},
	{
		"AsyncBatchDeleteLabelMembership", []uint{31},
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.AsyncBatchDeleteLabelMembershipFunc = func(ctx context.Context, batch [][2]uint) error { return nil }
			return &ds.AsyncBatchDeleteLabelMembershipFuncInvoked, func() error {
				return d.AsyncBatchDeleteLabelMembership(hookTestCtx, [][2]uint{{5, 31}})
			}
		},
	},
	{
		"AddLabelsToHost", []uint{41},
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.AddLabelsToHostFunc = func(ctx context.Context, hostID uint, labelIDs []uint) error { return nil }
			return &ds.AddLabelsToHostFuncInvoked, func() error { return d.AddLabelsToHost(hookTestCtx, 41, []uint{1}) }
		},
	},
	{
		"RemoveLabelsFromHost", []uint{51},
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.RemoveLabelsFromHostFunc = func(ctx context.Context, hostID uint, labelIDs []uint) error { return nil }
			return &ds.RemoveLabelsFromHostFuncInvoked, func() error { return d.RemoveLabelsFromHost(hookTestCtx, 51, []uint{1}) }
		},
	},
	{
		// The host-vitals label cron runs every 5 minutes for every
		// host-vitals label — it MUST invalidate only the hosts whose
		// membership value actually changed (returned by the datastore
		// method), never the deployment and never all members.
		"UpdateLabelMembershipByHostCriteria", []uint{61, 62},
		func(ds *mock.Store, d *Datastore) (*bool, func() error) {
			ds.UpdateLabelMembershipByHostCriteriaFunc = func(ctx context.Context, hvl fleet.HostVitalsLabel) (*fleet.Label, []uint, error) {
				return &fleet.Label{}, []uint{61, 62}, nil
			}
			return &ds.UpdateLabelMembershipByHostCriteriaFuncInvoked, func() error {
				_, _, err := d.UpdateLabelMembershipByHostCriteria(hookTestCtx, &fleet.Label{})
				return err
			}
		},
	},
}

func TestDeploymentHooksFireOnSuccess(t *testing.T) {

	for _, c := range deploymentHookCases {
		t.Run(c.name, func(t *testing.T) {
			ds, store, d := newTestDatastore()
			invoked, run := c.setupAndRun(ds, d)

			require.NoError(t, run())
			require.True(t, *invoked, "must delegate to the wrapped datastore")
			require.Equal(t, 1, store.invalidateCalls, "must invalidate on success")
			require.Empty(t, store.hostInvalidations, "deployment hooks must not fire per-host invalidations")
			wantLegacy, wantScopes := 0, 0
			switch c.resets {
			case resetsLegacyFlag:
				wantLegacy = 1
			case resetsLabelScopes:
				wantScopes = 1
			}
			require.Equal(t, wantLegacy, store.legacyResetCalls)
			require.Equal(t, wantScopes, store.labelScopesResetCalls)
		})
	}
}

// TestPerHostHooksFireOnSuccess exercises the routine label persistence
// paths and the direct membership mutations: they must fire PER-HOST
// invalidations only — never a deployment-wide invalidation (the HARD RULE:
// routine label traffic must never arm the deployment write fence).
func TestPerHostHooksFireOnSuccess(t *testing.T) {

	for _, c := range perHostHookCases {
		t.Run(c.name, func(t *testing.T) {
			ds, store, d := newTestDatastore()
			invoked, run := c.setupAndRun(ds, d)

			require.NoError(t, run())
			require.True(t, *invoked, "must delegate to the wrapped datastore")
			require.ElementsMatch(t, c.wantHosts, store.hostInvalidations)
			// HARD RULE routine label persistence must never arm the
			// deployment write fence.
			require.Equal(t, 0, store.invalidateCalls,
				"per-host hooks must never fire a deployment-wide invalidation")
			require.Equal(t, 0, store.legacyResetCalls)
			require.Equal(t, 0, store.labelScopesResetCalls)
		})
	}
}

// TestAsyncBatchUpdateLabelTimestampNotHooked: timestamp-only touches never
// change membership and must not invalidate anything.
func TestAsyncBatchUpdateLabelTimestampNotHooked(t *testing.T) {
	ctx := context.Background()
	ds, store, d := newTestDatastore()
	ds.AsyncBatchUpdateLabelTimestampFunc = func(ctx context.Context, ids []uint, ts time.Time) error { return nil }

	require.NoError(t, d.AsyncBatchUpdateLabelTimestamp(ctx, []uint{1, 2}, time.Now()))
	require.Empty(t, store.hostInvalidations)
	require.Equal(t, 0, store.invalidateCalls)
}

// TestNoInvalidationOnWriteError: a failed datastore write must NOT fire an
// invalidation (nothing changed), and the error must pass through.
func TestNoInvalidationOnWriteError(t *testing.T) {
	ctx := context.Background()
	ds, store, d := newTestDatastore()

	wantErr := errors.New("boom")
	ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error { return wantErr }
	ds.RecordLabelQueryExecutionsFunc = func(ctx context.Context, host *fleet.Host, results map[uint]*bool, t time.Time, deferred bool) error {
		return wantErr
	}
	ds.ApplyLabelSpecsWithAuthorFunc = func(ctx context.Context, specs []*fleet.LabelSpec, authorID *uint) error {
		return wantErr
	}

	err := d.SaveAppConfig(ctx, &fleet.AppConfig{})
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, 0, store.invalidateCalls)

	err = d.RecordLabelQueryExecutions(ctx, &fleet.Host{ID: 1}, nil, time.Now(), false)
	require.ErrorIs(t, err, wantErr)
	require.Empty(t, store.hostInvalidations)

	// failed label-spec persistence must not invalidate either
	err = d.ApplyLabelSpecsWithAuthor(ctx, []*fleet.LabelSpec{{Name: "l"}}, new(uint(1)))
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, 0, store.invalidateCalls)
	require.Equal(t, 0, store.labelScopesResetCalls)
}

// TestInvalidationErrorsAreSwallowed: Redis being down must never fail a
// datastore write — the ETag machinery fails open and the backstop TTLs
// bound any resulting staleness.
func TestInvalidationErrorsAreSwallowed(t *testing.T) {
	ctx := context.Background()
	ds, store, d := newTestDatastore()
	store.invalidateErr = errors.New("redis down")
	store.resetErr = errors.New("redis down")
	store.hostErr = errors.New("redis down")

	ds.SavePackFunc = func(ctx context.Context, pack *fleet.Pack) error { return nil }
	ds.RecordLabelQueryExecutionsFunc = func(ctx context.Context, host *fleet.Host, results map[uint]*bool, t time.Time, deferred bool) error {
		return nil
	}

	require.NoError(t, d.SavePack(ctx, &fleet.Pack{}))
	require.Equal(t, 1, store.invalidateCalls)
	require.Equal(t, 1, store.legacyResetCalls)

	require.NoError(t, d.RecordLabelQueryExecutions(ctx, &fleet.Host{ID: 9}, nil, time.Now(), false))
	require.Equal(t, []uint{9}, store.hostInvalidations)
}
