package etag_invalidate

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

type stubStore struct {
	fleet.ConfigETagStore
	invalidateCalls int
	resetCalls      int
	invalidateErr   error
	resetErr        error
}

func (s *stubStore) Invalidate(ctx context.Context) error {
	s.invalidateCalls++
	return s.invalidateErr
}

func (s *stubStore) ResetShortCircuitBlockedFlag(ctx context.Context) error {
	s.resetCalls++
	return s.resetErr
}

func newTestDatastore() (*mock.Store, *stubStore, *Datastore) {
	ds := new(mock.Store)
	store := &stubStore{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return ds, store, New(ds, store, logger)
}

// TestHooksFireOnSuccess exercises every overridden method: the write must
// delegate to the wrapped datastore and fire an invalidation on success.
// Pack and query methods must additionally reset the "short circuit blocked"
// gate flag (they can add/remove 2017 packs and label-scoped reports,
// respectively).
//
// ██ If you add a method to the etag_invalidate decorator, add a case here.
func TestHooksFireOnSuccess(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name        string
		resetsFlag  bool // pack and query CRUD must also reset the blocked gate flag
		setupAndRun func(ds *mock.Store, d *Datastore) (invoked *bool, run func() error)
	}{
		{
			"SaveAppConfig", false,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error { return nil }
				return &ds.SaveAppConfigFuncInvoked, func() error { return d.SaveAppConfig(ctx, &fleet.AppConfig{}) }
			},
		},
		{
			"NewTeam", false,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) { return team, nil }
				return &ds.NewTeamFuncInvoked, func() error { _, err := d.NewTeam(ctx, &fleet.Team{}); return err }
			},
		},
		{
			"SaveTeam", false,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) { return team, nil }
				return &ds.SaveTeamFuncInvoked, func() error { _, err := d.SaveTeam(ctx, &fleet.Team{}); return err }
			},
		},
		{
			"DeleteTeam", false,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.DeleteTeamFunc = func(ctx context.Context, tid uint) error { return nil }
				return &ds.DeleteTeamFuncInvoked, func() error { return d.DeleteTeam(ctx, 1) }
			},
		},
		{
			"ApplyQueries", true,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.ApplyQueriesFunc = func(ctx context.Context, authorID uint, queries []*fleet.Query, discard map[uint]struct{}) error { return nil }
				return &ds.ApplyQueriesFuncInvoked, func() error { return d.ApplyQueries(ctx, 1, nil, nil) }
			},
		},
		{
			"NewQuery", true,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) { return query, nil }
				return &ds.NewQueryFuncInvoked, func() error { _, err := d.NewQuery(ctx, &fleet.Query{}); return err }
			},
		},
		{
			"SaveQuery", true,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.SaveQueryFunc = func(ctx context.Context, query *fleet.Query, discard, deleteStats bool) error { return nil }
				return &ds.SaveQueryFuncInvoked, func() error { return d.SaveQuery(ctx, &fleet.Query{}, false, false) }
			},
		},
		{
			"DeleteQuery", true,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.DeleteQueryFunc = func(ctx context.Context, teamID *uint, name string) error { return nil }
				return &ds.DeleteQueryFuncInvoked, func() error { return d.DeleteQuery(ctx, nil, "q") }
			},
		},
		{
			"DeleteQueries", true,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.DeleteQueriesFunc = func(ctx context.Context, ids []uint) (uint, error) { return 1, nil }
				return &ds.DeleteQueriesFuncInvoked, func() error { _, err := d.DeleteQueries(ctx, []uint{1}); return err }
			},
		},
		{
			"ApplyPackSpecs", true,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.ApplyPackSpecsFunc = func(ctx context.Context, specs []*fleet.PackSpec) error { return nil }
				return &ds.ApplyPackSpecsFuncInvoked, func() error { return d.ApplyPackSpecs(ctx, nil) }
			},
		},
		{
			"NewPack", true,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.NewPackFunc = func(ctx context.Context, pack *fleet.Pack, opts ...fleet.OptionalArg) (*fleet.Pack, error) { return pack, nil }
				return &ds.NewPackFuncInvoked, func() error { _, err := d.NewPack(ctx, &fleet.Pack{}); return err }
			},
		},
		{
			"SavePack", true,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.SavePackFunc = func(ctx context.Context, pack *fleet.Pack) error { return nil }
				return &ds.SavePackFuncInvoked, func() error { return d.SavePack(ctx, &fleet.Pack{}) }
			},
		},
		{
			"DeletePack", true,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.DeletePackFunc = func(ctx context.Context, name string) error { return nil }
				return &ds.DeletePackFuncInvoked, func() error { return d.DeletePack(ctx, "p") }
			},
		},
		{
			"NewScheduledQuery", false,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.NewScheduledQueryFunc = func(ctx context.Context, sq *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
					return sq, nil
				}
				return &ds.NewScheduledQueryFuncInvoked, func() error { _, err := d.NewScheduledQuery(ctx, &fleet.ScheduledQuery{}); return err }
			},
		},
		{
			"SaveScheduledQuery", false,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.SaveScheduledQueryFunc = func(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) { return sq, nil }
				return &ds.SaveScheduledQueryFuncInvoked, func() error { _, err := d.SaveScheduledQuery(ctx, &fleet.ScheduledQuery{}); return err }
			},
		},
		{
			"DeleteScheduledQuery", false,
			func(ds *mock.Store, d *Datastore) (*bool, func() error) {
				ds.DeleteScheduledQueryFunc = func(ctx context.Context, id uint) error { return nil }
				return &ds.DeleteScheduledQueryFuncInvoked, func() error { return d.DeleteScheduledQuery(ctx, 1) }
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds, store, d := newTestDatastore()
			invoked, run := c.setupAndRun(ds, d)

			require.NoError(t, run())
			require.True(t, *invoked, "must delegate to the wrapped datastore")
			require.Equal(t, 1, store.invalidateCalls, "must invalidate on success")
			wantResets := 0
			if c.resetsFlag {
				wantResets = 1
			}
			require.Equal(t, wantResets, store.resetCalls)
		})
	}
}

// TestNoInvalidationOnWriteError: a failed datastore write must NOT fire an
// invalidation (nothing changed), and the error must pass through.
func TestNoInvalidationOnWriteError(t *testing.T) {
	ctx := context.Background()
	ds, store, d := newTestDatastore()

	wantErr := errors.New("boom")
	ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error { return wantErr }

	err := d.SaveAppConfig(ctx, &fleet.AppConfig{})
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, 0, store.invalidateCalls)
}

// TestInvalidationErrorsAreSwallowed: Redis being down must never fail a
// datastore write — the ETag machinery fails open and the backstop TTL
// bounds any resulting staleness.
func TestInvalidationErrorsAreSwallowed(t *testing.T) {
	ctx := context.Background()
	ds, store, d := newTestDatastore()
	store.invalidateErr = errors.New("redis down")
	store.resetErr = errors.New("redis down")

	ds.SavePackFunc = func(ctx context.Context, pack *fleet.Pack) error { return nil }

	require.NoError(t, d.SavePack(ctx, &fleet.Pack{}))
	require.Equal(t, 1, store.invalidateCalls)
	require.Equal(t, 1, store.resetCalls)
}
