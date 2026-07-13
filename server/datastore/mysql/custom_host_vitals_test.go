package mysql

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
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
		{"SetHostValueResendsReferencingProfiles", testSetHostCustomHostVitalValueResendsProfiles},
		{"ReconcileSnapshotMarksVitalDeclarations", testReconcileSnapshotMarksVitalDeclarations},
		{"ValidateReferencedCustomHostVitalsRejectsMalformed", testValidateReferencedCustomHostVitalsRejectsMalformed},
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
		require.Equal(t, "Unassigned", useErr.Entity.FleetName)

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
		require.Equal(t, "Unassigned", useErr.Entity.FleetName)

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

// Setting a host's value for a vital must re-queue the MDM profiles and DDM
// declarations already delivered to that host that reference the vital, so the
// reconcilers re-expand $FLEET_HOST_VITAL_<id> with the new value. Profiles
// referencing a different vital (or none), and other hosts, must be untouched.
func testSetHostCustomHostVitalValueResendsProfiles(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	host := test.NewHost(t, ds, "mac", "1", "mackey", "macuuid", time.Now())
	winHost := test.NewHost(t, ds, "win", "2", "winkey", "winuuid", time.Now(), test.WithPlatform("windows"))

	vitalID := createCustomHostVital(t, ds, "FUNCTION")
	otherID := createCustomHostVital(t, ds, "OTHER")
	token := fmt.Sprintf("$%s%d", fleet.CustomHostVitalPrefix, vitalID)
	otherToken := fmt.Sprintf("$%s%d", fleet.CustomHostVitalPrefix, otherID)

	// generateAppleCP/generateWindowsCP embed name+identifier in the profile
	// body, so passing the token there puts the reference in the content the
	// resend scan matches on.
	profVital, err := ds.NewMDMAppleConfigProfile(ctx, *generateAppleCP("pv", token, 0), nil)
	require.NoError(t, err)
	profOther, err := ds.NewMDMAppleConfigProfile(ctx, *generateAppleCP("po", otherToken, 0), nil)
	require.NoError(t, err)
	profNone, err := ds.NewMDMAppleConfigProfile(ctx, *generateAppleCP("pn", "plain", 0), nil)
	require.NoError(t, err)

	profWVital, err := ds.NewMDMWindowsConfigProfile(ctx, *generateWindowsCP("wv", token, 0), nil)
	require.NoError(t, err)
	profWNone, err := ds.NewMDMWindowsConfigProfile(ctx, *generateWindowsCP("wn", "plain", 0), nil)
	require.NoError(t, err)

	forceSetAppleHostProfileStatus(t, ds, host.UUID, profVital, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host.UUID, profOther, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host.UUID, profNone, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetWindowsHostProfileStatus(t, ds, winHost.UUID, profWVital, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetWindowsHostProfileStatus(t, ds, winHost.UUID, profWNone, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)

	// DDM declaration referencing the vital, delivered (verifying) to the mac host.
	bracedToken := fmt.Sprintf("${%s%d}", fleet.CustomHostVitalPrefix, vitalID)
	declVital, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier: "decl-vital", Name: "decl-vital",
		RawJSON: json.RawMessage(fmt.Sprintf(`{"note":"%s"}`, bracedToken)),
	}, nil)
	require.NoError(t, err)
	forceSetAppleHostDeclarationStatus(t, ds, host.UUID, declVital, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)

	// Set the value on both hosts; only referencing entities on the same host reset.
	require.NoError(t, ds.SetHostCustomHostVitalValue(ctx, host.ID, vitalID, "Engineering"))
	require.NoError(t, ds.SetHostCustomHostVitalValue(ctx, winHost.ID, vitalID, "Engineering"))

	// A reset row has NULL status, which assertHostProfileStatus reports as
	// pending. The declaration surfaces through GetHostMDMAppleProfiles too, so
	// it's included here; it should also be reset.
	assertHostProfileStatus(t, ds, host.UUID,
		hostProfileStatus{profVital.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profOther.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{declVital.DeclarationUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, winHost.UUID,
		hostProfileStatus{profWVital.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profWNone.ProfileUUID, fleet.MDMDeliveryVerifying})

	var declStatus *fleet.MDMDeliveryStatus
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &declStatus,
			`SELECT status FROM host_mdm_apple_declarations WHERE host_uuid = ? AND declaration_uuid = ?`,
			host.UUID, declVital.DeclarationUUID)
	})
	require.Nil(t, declStatus, "declaration should be reset (NULL status) so the DDM reconciler re-delivers it")
}

// The DDM reconcile snapshot must flag declarations that reference a custom host
// vital as HasFleetVariables, so the reconciler stamps variables_updated_at on
// the host declaration row — the signal handleDeclarationItems relies on to load
// raw_json, drop unresolvable declarations from the manifest, and cache-bust the
// DDM token. Custom host vitals aren't in mdm_configuration_profile_variables, so
// this depends on the body scan rather than the variables join.
func testReconcileSnapshotMarksVitalDeclarations(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// A macOS host must be MDM-enrolled to enter the reconcile window; otherwise
	// the snapshot skips loading declarations entirely.
	host := test.NewHost(t, ds, "macos-1", "1", "macos-1-key", "macos-1-uuid", time.Now())
	nanoEnroll(t, ds, host, false)

	vitalID := createCustomHostVital(t, ds, "FUNCTION")
	bracedToken := fmt.Sprintf("${%s%d}", fleet.CustomHostVitalPrefix, vitalID)

	declVital, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier: "decl-vital", Name: "decl-vital",
		RawJSON: json.RawMessage(fmt.Sprintf(`{"note":"%s"}`, bracedToken)),
	}, nil)
	require.NoError(t, err)
	declPlain, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier: "decl-plain", Name: "decl-plain",
		RawJSON: json.RawMessage(`{"note":"static"}`),
	}, nil)
	require.NoError(t, err)

	_, allDecls, _, _, err := ds.GetAppleDeclarationReconcileSnapshot(ctx, "", 100)
	require.NoError(t, err)

	byUUID := make(map[string]*fleet.AppleDeclarationForReconcile, len(allDecls))
	for _, d := range allDecls {
		byUUID[d.DeclarationUUID] = d
	}
	vitalDecl := byUUID[declVital.DeclarationUUID]
	plainDecl := byUUID[declPlain.DeclarationUUID]
	require.NotNil(t, vitalDecl, "vital declaration missing from reconcile snapshot")
	require.NotNil(t, plainDecl, "plain declaration missing from reconcile snapshot")
	assert.True(t, vitalDecl.HasFleetVariables, //nolint:nilaway // cannot be nil due to require.NotNil above
		"declaration referencing a custom host vital should be marked HasFleetVariables")
	assert.False(t, plainDecl.HasFleetVariables, //nolint:nilaway // cannot be nil due to require.NotNil above
		"declaration without any variables should not be marked")
}

// A $FLEET_HOST_VITAL_ token whose suffix isn't a valid vital ID must be rejected on upload
func testValidateReferencedCustomHostVitalsRejectsMalformed(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	vitalID := createCustomHostVital(t, ds, "Function")

	// Valid numeric reference to an existing vital passes.
	require.NoError(t, ds.ValidateReferencedCustomHostVitals(ctx,
		[]string{fmt.Sprintf("echo $%s%d", fleet.CustomHostVitalPrefix, vitalID)}))

	// Non-numeric suffix is a malformed reference -> InvalidCustomHostVitalRefError.
	var invalidErr *fleet.InvalidCustomHostVitalRefError
	err := ds.ValidateReferencedCustomHostVitals(ctx,
		[]string{fmt.Sprintf("echo $%sasset_tag", fleet.CustomHostVitalPrefix)})
	require.ErrorAs(t, err, &invalidErr)

	// Braced malformed form -> also InvalidCustomHostVitalRefError.
	invalidErr = nil
	err = ds.ValidateReferencedCustomHostVitals(ctx,
		[]string{fmt.Sprintf("echo ${%sFOOBAR}", fleet.CustomHostVitalPrefix)})
	require.ErrorAs(t, err, &invalidErr)

	// Numeric-but-nonexistent id -> MissingCustomHostVitalsError (distinct from malformed).
	var missingErr *fleet.MissingCustomHostVitalsError
	err = ds.ValidateReferencedCustomHostVitals(ctx,
		[]string{fmt.Sprintf("echo $%s999999", fleet.CustomHostVitalPrefix)})
	require.ErrorAs(t, err, &missingErr)

	// No token at all -> no error.
	require.NoError(t, ds.ValidateReferencedCustomHostVitals(ctx, []string{"echo hello"}))
}
