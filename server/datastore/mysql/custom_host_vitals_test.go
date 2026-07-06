package mysql

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
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
		{"DeleteUsedCustomHostVital", testDeleteUsedCustomHostVital},
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

	// No values yet.
	got, err := ds.GetHostCustomHostVitals(ctx, host.ID)
	require.NoError(t, err)
	require.Empty(t, got)

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

func testDeleteUsedCustomHostVital(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	foobarTeam, err := ds.NewTeam(ctx, &fleet.Team{Name: "Foobar"})
	require.NoError(t, err)

	id := createCustomHostVital(t, ds, "FUNCTION")
	id2 := createCustomHostVital(t, ds, "OTHER")

	// $FLEET_HOST_VITAL_<id> token that references FUNCTION.
	token := fmt.Sprintf("$%s%d", fleet.CustomHostVitalPrefix, id)
	// ${FLEET_HOST_VITAL_<id>} braced form.
	bracedToken := fmt.Sprintf("${%s%d}", fleet.CustomHostVitalPrefix, id)

	t.Run("apple configuration profiles", func(t *testing.T) {
		appleProfile, err := ds.NewMDMAppleConfigProfile(ctx, fleet.MDMAppleConfigProfile{
			Name:         "Name0",
			Identifier:   "Identifier0",
			Mobileconfig: []byte(token),
		}, nil)
		require.NoError(t, err)

		_, err = ds.DeleteCustomHostVital(ctx, id)
		require.Error(t, err)
		var useErr *fleet.CustomHostVitalUsedError
		require.ErrorAs(t, err, &useErr)
		require.Equal(t, id, useErr.CustomHostVitalID)
		require.Equal(t, "FUNCTION", useErr.CustomHostVitalName)
		require.Equal(t, "apple_profile", useErr.Entity.Type)
		require.Equal(t, "Name0", useErr.Entity.Name)
		require.Equal(t, "No team", useErr.Entity.FleetName)

		// Deleting an unreferenced vital is allowed.
		_, err = ds.DeleteCustomHostVital(ctx, id2)
		require.NoError(t, err)
		// Recreate for later subtests.
		id2 = createCustomHostVital(t, ds, "OTHER")

		require.NoError(t, ds.DeleteMDMAppleConfigProfile(ctx, appleProfile.ProfileUUID))
	})

	t.Run("apple declarations", func(t *testing.T) {
		decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Identifier: "decl-1",
			Name:       "decl-1",
			RawJSON:    json.RawMessage(fmt.Sprintf(`{"Identifier": "%s"}`, bracedToken)),
			TeamID:     &foobarTeam.ID,
		}, nil)
		require.NoError(t, err)

		_, err = ds.DeleteCustomHostVital(ctx, id)
		require.Error(t, err)
		var useErr *fleet.CustomHostVitalUsedError
		require.ErrorAs(t, err, &useErr)
		require.Equal(t, id, useErr.CustomHostVitalID)
		require.Equal(t, "FUNCTION", useErr.CustomHostVitalName)
		require.Equal(t, "apple_declaration", useErr.Entity.Type)
		require.Equal(t, "decl-1", useErr.Entity.Name)
		require.Equal(t, "Foobar", useErr.Entity.FleetName)

		require.NoError(t, ds.DeleteMDMAppleDeclaration(ctx, decl.DeclarationUUID))
	})

	t.Run("windows profiles", func(t *testing.T) {
		winProfile, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
			Name:   "zoo",
			SyncML: []byte(fmt.Sprintf("<Replace>%s</Replace>", token)),
		}, nil)
		require.NoError(t, err)

		_, err = ds.DeleteCustomHostVital(ctx, id)
		require.Error(t, err)
		var useErr *fleet.CustomHostVitalUsedError
		require.ErrorAs(t, err, &useErr)
		require.Equal(t, id, useErr.CustomHostVitalID)
		require.Equal(t, "FUNCTION", useErr.CustomHostVitalName)
		require.Equal(t, "windows_profile", useErr.Entity.Type)
		require.Equal(t, "zoo", useErr.Entity.Name)
		require.Equal(t, "No team", useErr.Entity.FleetName)

		require.NoError(t, ds.DeleteMDMWindowsConfigProfile(ctx, winProfile.ProfileUUID))
	})

	t.Run("scripts", func(t *testing.T) {
		script, err := ds.NewScript(ctx, &fleet.Script{
			Name:           "collect.sh",
			ScriptContents: fmt.Sprintf("echo %s", token),
			TeamID:         &foobarTeam.ID,
		})
		require.NoError(t, err)

		_, err = ds.DeleteCustomHostVital(ctx, id)
		require.Error(t, err)
		var useErr *fleet.CustomHostVitalUsedError
		require.ErrorAs(t, err, &useErr)
		require.Equal(t, id, useErr.CustomHostVitalID)
		require.Equal(t, "FUNCTION", useErr.CustomHostVitalName)
		require.Equal(t, "script", useErr.Entity.Type)
		require.Equal(t, "collect.sh", useErr.Entity.Name)
		require.Equal(t, "Foobar", useErr.Entity.FleetName)

		require.NoError(t, ds.DeleteScript(ctx, script.ID))
	})

	t.Run("software installers", func(t *testing.T) {
		user := test.NewUser(t, ds, "Installer Author", "chv-del-installer@example.com", true)
		tfr, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
		require.NoError(t, err)
		installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			InstallScript:   fmt.Sprintf("install %s", token),
			UninstallScript: "uninstall",
			InstallerFile:   tfr,
			StorageID:       "chv-del-storage",
			Filename:        "chv-del.pkg",
			Title:           "chv-del-title",
			Version:         "1.0",
			Source:          "apps",
			TeamID:          &foobarTeam.ID,
			UserID:          user.ID,
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		})
		require.NoError(t, err)

		_, err = ds.DeleteCustomHostVital(ctx, id)
		require.Error(t, err)
		var useErr *fleet.CustomHostVitalUsedError
		require.ErrorAs(t, err, &useErr)
		require.Equal(t, id, useErr.CustomHostVitalID)
		require.Equal(t, "FUNCTION", useErr.CustomHostVitalName)
		require.Equal(t, "software_installer", useErr.Entity.Type)
		require.Equal(t, "chv-del-title", useErr.Entity.Name)
		require.Equal(t, "Foobar", useErr.Entity.FleetName)

		require.NoError(t, ds.DeleteSoftwareInstaller(ctx, installerID))
	})

	t.Run("setup experience scripts", func(t *testing.T) {
		require.NoError(t, ds.SetSetupExperienceScript(ctx, &fleet.Script{
			Name:           "setup.sh",
			ScriptContents: fmt.Sprintf("echo %s", token),
			TeamID:         &foobarTeam.ID,
		}))

		_, err := ds.DeleteCustomHostVital(ctx, id)
		require.Error(t, err)
		var useErr *fleet.CustomHostVitalUsedError
		require.ErrorAs(t, err, &useErr)
		require.Equal(t, id, useErr.CustomHostVitalID)
		require.Equal(t, "FUNCTION", useErr.CustomHostVitalName)
		require.Equal(t, "setup_experience_script", useErr.Entity.Type)
		require.Equal(t, "setup.sh", useErr.Entity.Name)
		require.Equal(t, "Foobar", useErr.Entity.FleetName)

		require.NoError(t, ds.DeleteSetupExperienceScript(ctx, &foobarTeam.ID))
	})

	// With all references removed, delete now succeeds.
	name, err := ds.DeleteCustomHostVital(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "FUNCTION", name)
}
