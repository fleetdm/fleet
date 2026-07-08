package mysql

import (
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestCustomHostVitals(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"CreateCustomHostVital", testCreateCustomHostVital},
		{"ListCustomHostVitals", testListCustomHostVitals},
		{"UpdateCustomHostVital", testUpdateCustomHostVital},
		{"SetAndGetHostCustomHostVitals", testSetAndGetHostCustomHostVitals},
		{"GetCustomHostVitals", testGetCustomHostVitals},
		{"DeleteCustomHostVital", testDeleteCustomHostVital},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// createCustomHostVital is a test helper that creates a definition and returns its id.
func createCustomHostVital(t *testing.T, ds *Datastore, name string) uint {
	v, err := ds.CreateCustomHostVital(t.Context(), name)
	require.NoError(t, err)
	return v.ID
}

func testCreateCustomHostVital(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	vital, err := ds.CreateCustomHostVital(ctx, "Asset tag")
	require.NoError(t, err)
	require.NotZero(t, vital.ID)
	require.Equal(t, "Asset tag", vital.Name)

	// Duplicate name surfaces AlreadyExistsError.
	dup, err := ds.CreateCustomHostVital(ctx, "Asset tag")
	require.Error(t, err)
	var aee fleet.AlreadyExistsError
	require.ErrorAs(t, err, &aee)
	require.Zero(t, dup.ID)
}

func testListCustomHostVitals(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	funcID := createCustomHostVital(t, ds, "Function")
	deptID := createCustomHostVital(t, ds, "Department")

	list := func(opt fleet.ListOptions) []fleet.CustomHostVital {
		vitals, _, _, err := ds.ListCustomHostVitals(ctx, opt)
		require.NoError(t, err)
		return vitals
	}

	names := func(vitals []fleet.CustomHostVital) []string {
		out := make([]string, 0, len(vitals))
		for _, v := range vitals {
			out = append(out, v.Name)
		}
		return out
	}

	// No filter: both definitions returned.
	require.ElementsMatch(t, []string{"Function", "Department"}, names(list(fleet.ListOptions{})))

	// Count is returned.
	_, _, count, err := ds.ListCustomHostVitals(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Search by name (case-insensitive via the collation, substring).
	require.ElementsMatch(t, []string{"Function"}, names(list(fleet.ListOptions{MatchQuery: "func"})))
	require.ElementsMatch(t, []string{"Department"}, names(list(fleet.ListOptions{MatchQuery: "depart"})))

	// Search by the derived $FLEET_HOST_VITAL_<id> variable token. The token is
	// not stored; ListCustomHostVitals matches CONCAT('$FLEET_HOST_VITAL_', id).
	funcToken := fmt.Sprintf("$%s%d", fleet.CustomHostVitalPrefix, funcID)
	require.ElementsMatch(t, []string{"Function"}, names(list(fleet.ListOptions{MatchQuery: funcToken})))
	deptToken := fmt.Sprintf("$%s%d", fleet.CustomHostVitalPrefix, deptID)
	require.ElementsMatch(t, []string{"Department"}, names(list(fleet.ListOptions{MatchQuery: deptToken})))

	// A partial token prefix (the shared namespace) matches both.
	require.ElementsMatch(t, []string{"Function", "Department"},
		names(list(fleet.ListOptions{MatchQuery: "$" + fleet.CustomHostVitalPrefix})))

	// A token for a non-existent id matches nothing.
	require.Empty(t, list(fleet.ListOptions{MatchQuery: fmt.Sprintf("$%s999999", fleet.CustomHostVitalPrefix)}))
}

func testUpdateCustomHostVital(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	id := createCustomHostVital(t, ds, "Function")
	createCustomHostVital(t, ds, "Other")

	// Rename succeeds and returns the updated definition.
	updated, err := ds.UpdateCustomHostVital(ctx, id, "Role")
	require.NoError(t, err)
	require.Equal(t, id, updated.ID)
	require.Equal(t, "Role", updated.Name)
	vitals, err := ds.GetCustomHostVitals(ctx, []uint{id})
	require.NoError(t, err)
	require.Len(t, vitals, 1)
	require.Equal(t, "Role", vitals[0].Name)

	// No-op rename (same name) is not treated as NotFound.
	_, err = ds.UpdateCustomHostVital(ctx, id, "Role")
	require.NoError(t, err)

	// Renaming to an existing name surfaces AlreadyExistsError.
	_, err = ds.UpdateCustomHostVital(ctx, id, "Other")
	require.Error(t, err)
	var aee fleet.AlreadyExistsError
	require.ErrorAs(t, err, &aee)

	// Updating a non-existent id surfaces NotFoundError.
	_, err = ds.UpdateCustomHostVital(ctx, 999999, "Whatever")
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)
}

func testSetAndGetHostCustomHostVitals(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "chv-host",
		UUID:            "chv-host-uuid",
		OsqueryHostID:   new("chv-host-osquery-id"),
		NodeKey:         new("chv-host-node-key"),
		DetailUpdatedAt: time.Now(),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	funcID := createCustomHostVital(t, ds, "Function")
	deptID := createCustomHostVital(t, ds, "Department")

	// With no per-host values set yet, every definition is still returned for
	// the host with an empty value.
	got, err := ds.GetHostCustomHostVitals(ctx, host.ID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	for _, v := range got {
		require.Empty(t, v.Value)
	}

	// Insert two values.
	require.NoError(t, ds.SetHostCustomHostVitalValue(ctx, host.ID, funcID, "engineering"))
	require.NoError(t, ds.SetHostCustomHostVitalValue(ctx, host.ID, deptID, "R&D"))

	got, err = ds.GetHostCustomHostVitals(ctx, host.ID)
	require.NoError(t, err)
	byID := make(map[uint]fleet.HostCustomHostVital, len(got))
	for _, v := range got {
		byID[v.CustomHostVitalID] = v
	}
	require.Len(t, byID, 2)
	require.Equal(t, "Function", byID[funcID].Name)
	require.Equal(t, "engineering", byID[funcID].Value)
	require.Equal(t, "Department", byID[deptID].Name)
	require.Equal(t, "R&D", byID[deptID].Value)

	// Upsert overwrites the existing value for (host, vital).
	require.NoError(t, ds.SetHostCustomHostVitalValue(ctx, host.ID, funcID, "sales"))
	got, err = ds.GetHostCustomHostVitals(ctx, host.ID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	for _, v := range got {
		if v.CustomHostVitalID == funcID {
			require.Equal(t, "sales", v.Value)
		}
	}
}

func testGetCustomHostVitals(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	funcID := createCustomHostVital(t, ds, "Function")
	deptID := createCustomHostVital(t, ds, "Department")

	// Known ids resolve; an unknown id is silently omitted.
	got, err := ds.GetCustomHostVitals(ctx, []uint{funcID, deptID, 999999})
	require.NoError(t, err)
	names := make([]string, 0, len(got))
	for _, v := range got {
		names = append(names, v.Name)
	}
	require.ElementsMatch(t, []string{"Function", "Department"}, names)
}

func testDeleteCustomHostVital(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "chv-del-host",
		UUID:            "chv-del-host-uuid",
		OsqueryHostID:   new("chv-del-host-osquery-id"),
		NodeKey:         new("chv-del-host-node-key"),
		DetailUpdatedAt: time.Now(),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	id := createCustomHostVital(t, ds, "Function")
	require.NoError(t, ds.SetHostCustomHostVitalValue(ctx, host.ID, id, "engineering"))

	name, err := ds.DeleteCustomHostVital(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "Function", name)

	got, err := ds.GetHostCustomHostVitals(ctx, host.ID)
	require.NoError(t, err)
	require.Empty(t, got)

	// Deleting a non-existent id surfaces NotFoundError.
	_, err = ds.DeleteCustomHostVital(ctx, 999999)
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)
}
