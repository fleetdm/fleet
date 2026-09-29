package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestGetSoftwareTitleDisplayName(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"NoOverrideReturnsNil", testGetSoftwareTitleDisplayName_NoOverride},
		{"WithOverride", testGetSoftwareTitleDisplayName_WithOverride},
		{"PerTeamIsolation", testGetSoftwareTitleDisplayName_PerTeamIsolation},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// insertSoftwareTitleForTest creates a bare software_titles row so a display
// name override has something to point at.
func insertSoftwareTitleForTest(t *testing.T, ds *Datastore, name string) uint {
	t.Helper()
	ctx := context.Background()
	res, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO software_titles (name, source) VALUES (?, 'apps')`, name)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	require.Positive(t, id)
	return uint(id) //nolint:gosec // AUTO_INCREMENT id is always positive
}

func testGetSoftwareTitleDisplayName_NoOverride(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	titleID := insertSoftwareTitleForTest(t, ds, "Firefox")

	got, err := ds.GetSoftwareTitleDisplayName(ctx, nil, titleID)
	require.NoError(t, err)
	require.Nil(t, got, "no override should return nil, not an error")

	got, err = ds.GetSoftwareTitleDisplayName(ctx, new(uint(42)), titleID)
	require.NoError(t, err)
	require.Nil(t, got)
}

func testGetSoftwareTitleDisplayName_WithOverride(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	titleID := insertSoftwareTitleForTest(t, ds, "Firefox")

	// no-team override (stored as team_id=0)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return updateSoftwareTitleDisplayName(ctx, q, nil, titleID, "Mozilla Firefox (managed)")
	})

	got, err := ds.GetSoftwareTitleDisplayName(ctx, nil, titleID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "Mozilla Firefox (managed)", *got)
}

func testGetSoftwareTitleDisplayName_PerTeamIsolation(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	titleID := insertSoftwareTitleForTest(t, ds, "Firefox")

	teamA, err := ds.NewTeam(ctx, &fleet.Team{Name: "team-a"})
	require.NoError(t, err)
	teamB, err := ds.NewTeam(ctx, &fleet.Team{Name: "team-b"})
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return updateSoftwareTitleDisplayName(ctx, q, &teamA.ID, titleID, "A-rename")
	})

	got, err := ds.GetSoftwareTitleDisplayName(ctx, &teamA.ID, titleID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "A-rename", *got)

	// Team B has no override and MUST NOT inherit Team A's rename.
	got, err = ds.GetSoftwareTitleDisplayName(ctx, &teamB.ID, titleID)
	require.NoError(t, err)
	require.Nil(t, got)

	// No-team also doesn't see Team A's rename.
	got, err = ds.GetSoftwareTitleDisplayName(ctx, nil, titleID)
	require.NoError(t, err)
	require.Nil(t, got)
}
