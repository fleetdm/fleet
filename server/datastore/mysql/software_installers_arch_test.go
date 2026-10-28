package mysql

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestFleetMaintainedAppArchSiblings(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Versions", testArchSiblingsVersions},
		{"Routing", testArchSiblingsRouting},
		{"PatchPolicy", testArchSiblingsPatchPolicy},
		{"SetupExperience", testArchSiblingsSetupExperience},
		{"BatchSet", testArchSiblingsBatchSet},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

const chromeUpgradeCode = "{C1DFDF69-5945-32F2-A35E-EE94C99C7CF4}"

type archSiblingsFixture struct {
	user          *fleet.User
	team          *fleet.Team
	x64, arm64    *fleet.MaintainedApp
	upgradeCode   string
	installerArgs func(app *fleet.MaintainedApp, storage, version string) *fleet.UploadSoftwareInstallerPayload
}

func newArchSiblingsFixture(t *testing.T, ds *Datastore) *archSiblingsFixture {
	ctx := t.Context()
	f := &archSiblingsFixture{upgradeCode: chromeUpgradeCode}
	f.user = test.NewUser(t, ds, "Alice", "alice@example.com", true)
	var err error
	f.team, err = ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	f.x64, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name: "Google Chrome", Slug: "google-chrome/windows", Platform: "windows", UniqueIdentifier: "Google Chrome",
	})
	require.NoError(t, err)
	f.arm64, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name: "Google Chrome (ARM64)", Slug: "google-chrome-arm64/windows", Platform: "windows", UniqueIdentifier: "Google Chrome",
	})
	require.NoError(t, err)

	f.installerArgs = func(app *fleet.MaintainedApp, storage, version string) *fleet.UploadSoftwareInstallerPayload {
		tfr, err := fleet.NewTempFileReader(strings.NewReader(storage), t.TempDir)
		require.NoError(t, err)
		return &fleet.UploadSoftwareInstallerPayload{
			InstallerFile:        tfr,
			Extension:            "msi",
			StorageID:            storage,
			Filename:             storage + ".msi",
			Title:                "Google Chrome",
			UpgradeCode:          f.upgradeCode,
			Version:              version,
			Source:               "programs",
			Platform:             "windows",
			InstallScript:        "install " + storage,
			UninstallScript:      "uninstall " + storage,
			PatchQuery:           "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Google Chrome' AND version_compare(version, '" + version + "') < 0);",
			FleetMaintainedAppID: new(app.ID),
			FMAName:              app.Name,
			FMASlug:              app.Slug,
			UserID:               f.user.ID,
			ValidatedLabels:      &fleet.LabelIdentsWithScope{},
			TeamID:               &f.team.ID,
		}
	}
	return f
}

// addBoth adds the x64 and ARM64 builds at the same version and returns their installer IDs and the shared title.
func (f *archSiblingsFixture) addBoth(t *testing.T, ds *Datastore) (x64ID, arm64ID, titleID uint) {
	ctx := t.Context()
	x64ID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, f.installerArgs(f.x64, "chrome-x64-1", "1.0"))
	require.NoError(t, err)
	arm64ID, armTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, f.installerArgs(f.arm64, "chrome-arm64-1", "1.0"))
	require.NoError(t, err)
	require.Equal(t, titleID, armTitleID)
	return x64ID, arm64ID, titleID
}

func activeInstallerIDs(t *testing.T, ds *Datastore, teamID, titleID uint) []uint {
	var ids []uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(t.Context(), q, &ids,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND title_id = ? AND is_active = 1 ORDER BY id`, teamID, titleID)
	})
	return ids
}

func newWindowsHostWithCPU(t *testing.T, ds *Datastore, teamID uint, cpuType string) *fleet.Host {
	ctx := t.Context()
	id := uuid.NewString()
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "win-" + id,
		OsqueryHostID:   new(id),
		NodeKey:         new(id),
		UUID:            id,
		Platform:        "windows",
		CPUType:         cpuType,
		TeamID:          &teamID,
		LastEnrolledAt:  time.Now(),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
	})
	require.NoError(t, err)
	// NewHost doesn't write cpu_type; osquery's system_info detail query does.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE hosts SET cpu_type = ? WHERE id = ?`, cpuType, host.ID)
		return err
	})
	host.CPUType = cpuType
	return host
}

func testArchSiblingsVersions(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newArchSiblingsFixture(t, ds)
	x64ID, arm64ID, titleID := f.addBoth(t, ds)
	require.Equal(t, []uint{x64ID, arm64ID}, activeInstallerIDs(t, ds, f.team.ID, titleID))

	candidates, err := ds.ListFleetMaintainedAppActiveInstallers(ctx)
	require.NoError(t, err)
	var onTitle []uint
	for _, c := range candidates {
		if c.TitleID == titleID {
			onTitle = append(onTitle, c.FleetMaintainedAppID)
		}
	}
	require.ElementsMatch(t, []uint{f.x64.ID, f.arm64.ID}, onTitle)

	// A policy installs the ARM64 build.
	policy, err := ds.NewTeamPolicy(ctx, f.team.ID, nil, fleet.PolicyPayload{
		Name: "Chrome ARM64", Query: "SELECT 1;", Platform: "windows", SoftwareInstallerID: &arm64ID,
	})
	require.NoError(t, err)

	// A new x64 version, cached and promoted with a pin, leaves the ARM64 build and its policy alone.
	x64v2 := f.installerArgs(f.x64, "chrome-x64-2", "2.0")
	x64v2ID, err := ds.InsertFleetMaintainedAppVersion(ctx, x64ID, x64v2)
	require.NoError(t, err)
	require.NoError(t, ds.SetFleetMaintainedAppActiveInstaller(ctx, &fleet.UpdateSoftwareInstallerPayload{
		TeamID: &f.team.ID, TitleID: titleID, PinnedVersion: new("2.0"),
	}, x64v2ID))
	require.Equal(t, []uint{arm64ID, x64v2ID}, activeInstallerIDs(t, ds, f.team.ID, titleID))
	policy, err = ds.Policy(ctx, policy.ID)
	require.NoError(t, err)
	require.Equal(t, arm64ID, *policy.SoftwareInstallerID)

	pin, err := ds.GetPinnedVersion(ctx, &f.team.ID, titleID, f.x64.ID)
	require.NoError(t, err)
	require.Equal(t, "2.0", *pin)
	_, err = ds.GetPinnedVersion(ctx, &f.team.ID, titleID, f.arm64.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// Each build lists its own cached versions; the ARM64 build can cache the x64 build's version too.
	arm64v2ID, err := ds.InsertFleetMaintainedAppVersion(ctx, arm64ID, f.installerArgs(f.arm64, "chrome-arm64-2", "2.0"))
	require.NoError(t, err)
	require.NotEqual(t, x64v2ID, arm64v2ID)
	versions := func(fmaID uint) []string {
		vs, err := ds.GetFleetMaintainedVersionsByTitleID(ctx, &f.team.ID, titleID, fmaID)
		require.NoError(t, err)
		return versionStrings(vs)
	}
	require.Equal(t, []string{"2.0", "1.0"}, versions(f.x64.ID))
	require.Equal(t, []string{"2.0", "1.0"}, versions(f.arm64.ID))

	// Promoting the ARM64 build moves its policy to the new version only.
	require.NoError(t, ds.SetFleetMaintainedAppActiveInstaller(ctx, &fleet.UpdateSoftwareInstallerPayload{
		TeamID: &f.team.ID, TitleID: titleID,
	}, arm64v2ID))
	require.Equal(t, []uint{x64v2ID, arm64v2ID}, activeInstallerIDs(t, ds, f.team.ID, titleID))
	policy, err = ds.Policy(ctx, policy.ID)
	require.NoError(t, err)
	require.Equal(t, arm64v2ID, *policy.SoftwareInstallerID)

	// Caching a third version evicts only that build's oldest one.
	_, err = ds.InsertFleetMaintainedAppVersion(ctx, x64v2ID, f.installerArgs(f.x64, "chrome-x64-3", "3.0"))
	require.NoError(t, err)
	require.Equal(t, []string{"3.0", "2.0"}, versions(f.x64.ID))
	require.Equal(t, []string{"2.0", "1.0"}, versions(f.arm64.ID))
}

func testArchSiblingsRouting(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newArchSiblingsFixture(t, ds)
	x64ID, arm64ID, titleID := f.addBoth(t, ds)

	armHost := newWindowsHostWithCPU(t, ds, f.team.ID, "ARM")
	x64Host := newWindowsHostWithCPU(t, ds, f.team.ID, "x86_64")
	unknownHost := newWindowsHostWithCPU(t, ds, f.team.ID, "")

	resolve := func(installerID uint, host *fleet.Host, opts fleet.HostSoftwareInstallOptions) (uint, bool) {
		id, ok, err := ds.ResolveFleetMaintainedAppBuildForHost(ctx, installerID, host, opts)
		require.NoError(t, err)
		return id, ok
	}
	for _, c := range []struct {
		installerID uint
		host        *fleet.Host
		want        uint
	}{
		{x64ID, armHost, arm64ID},
		{arm64ID, armHost, arm64ID},
		{x64ID, x64Host, x64ID},
		{arm64ID, x64Host, x64ID},
		{arm64ID, unknownHost, x64ID},
	} {
		id, ok := resolve(c.installerID, c.host, fleet.HostSoftwareInstallOptions{})
		require.True(t, ok)
		require.Equal(t, c.want, id, "installer %d on %q", c.installerID, c.host.CPUType)
	}

	displayed := func(host *fleet.Host) (uint, bool) {
		resolved, inScope, err := ds.resolveFirstAddedInstallersForHost(ctx, host, []uint{titleID})
		require.NoError(t, err)
		_, ok := inScope[titleID]
		return resolved[titleID].ID, ok
	}
	id, ok := displayed(armHost)
	require.True(t, ok)
	require.Equal(t, arm64ID, id)
	id, ok = displayed(x64Host)
	require.True(t, ok)
	require.Equal(t, x64ID, id)
	id, ok = displayed(unknownHost)
	require.True(t, ok)
	require.Equal(t, x64ID, id)
	// The admin host software list loads the host with HostLite.
	liteHost, err := ds.HostLite(ctx, armHost.ID)
	require.NoError(t, err)
	id, ok = displayed(liteHost)
	require.True(t, ok)
	require.Equal(t, arm64ID, id)

	// Labels still apply. Scope each build to a label no host is in.
	label, err := ds.NewLabel(ctx, &fleet.Label{Name: t.Name(), Query: "SELECT 1", LabelMembershipType: fleet.LabelMembershipTypeManual})
	require.NoError(t, err)
	scopeToLabel := func(installerID uint) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `INSERT INTO software_installer_labels (software_installer_id, label_id, exclude, require_all) VALUES (?, ?, 0, 0)`, installerID, label.ID)
			return err
		})
	}

	// An ARM host out of scope for the ARM64 build gets the x64 build, which runs under emulation,
	// whichever build the caller started from.
	scopeToLabel(arm64ID)
	id, ok = resolve(x64ID, armHost, fleet.HostSoftwareInstallOptions{})
	require.True(t, ok)
	require.Equal(t, x64ID, id)
	id, ok = resolve(arm64ID, armHost, fleet.HostSoftwareInstallOptions{})
	require.True(t, ok)
	require.Equal(t, x64ID, id)
	id, ok = displayed(armHost)
	require.True(t, ok)
	require.Equal(t, x64ID, id)

	// An x64 host out of scope for the x64 build gets nothing: the ARM64 build can't run there.
	scopeToLabel(x64ID)
	_, ok = resolve(arm64ID, x64Host, fleet.HostSoftwareInstallOptions{})
	require.False(t, ok)
	_, ok = displayed(x64Host)
	require.False(t, ok)

	// Setup experience ignores labels and goes by which builds are selected for it.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE software_installers SET install_during_setup = 1 WHERE id = ?`, arm64ID)
		return err
	})
	id, ok = resolve(x64ID, armHost, fleet.HostSoftwareInstallOptions{ForSetupExperience: true})
	require.True(t, ok)
	require.Equal(t, arm64ID, id)
}

func testArchSiblingsPatchPolicy(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newArchSiblingsFixture(t, ds)
	x64ID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, f.installerArgs(f.x64, "chrome-x64-1", "1.0"))
	require.NoError(t, err)

	policy, err := ds.NewTeamPolicy(ctx, f.team.ID, nil, fleet.PolicyPayload{Type: fleet.PolicyTypePatch, PatchSoftwareTitleID: &titleID})
	require.NoError(t, err)
	x64Query := policy.Query
	require.NotContains(t, x64Query, "cpu_type")

	// Regenerating an unchanged query keeps the hosts' results.
	host := test.NewHost(t, ds, "patch-host", "", "patch-host-key", "patch-host-uuid", time.Now())
	_, err = ds.RecordPolicyQueryExecutions(ctx, host, map[uint]*bool{policy.ID: new(false)}, time.Now(), false, nil)
	require.NoError(t, err)
	membership := func() int {
		var n int
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &n, `SELECT COUNT(*) FROM policy_membership WHERE policy_id = ?`, policy.ID)
		})
		return n
	}
	require.Equal(t, 1, membership())
	require.NoError(t, ds.RegeneratePatchPolicyQuery(ctx, &f.team.ID, titleID))
	require.Equal(t, 1, membership())

	// Adding the ARM64 build at a lagging version splits the query by CPU and resets the results.
	arm64ID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, f.installerArgs(f.arm64, "chrome-arm64-0", "0.9"))
	require.NoError(t, err)
	require.NoError(t, ds.RegeneratePatchPolicyQuery(ctx, &f.team.ID, titleID))
	policy, err = ds.Policy(ctx, policy.ID)
	require.NoError(t, err)
	require.Contains(t, policy.Query, "cpu_type LIKE 'ARM%'")
	require.Contains(t, policy.Query, "'1.0'")
	require.Contains(t, policy.Query, "'0.9'")
	require.Zero(t, membership())

	// One build can be deleted while the other backs the patch policy, which then covers it alone.
	require.NoError(t, ds.DeleteSoftwareInstaller(ctx, arm64ID))
	require.NoError(t, ds.RegeneratePatchPolicyQuery(ctx, &f.team.ID, titleID))
	policy, err = ds.Policy(ctx, policy.ID)
	require.NoError(t, err)
	require.Equal(t, x64Query, policy.Query)

	// The last build can't be deleted under a patch policy.
	require.ErrorContains(t, ds.DeleteSoftwareInstaller(ctx, x64ID), "patch policy")
}

func testArchSiblingsSetupExperience(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newArchSiblingsFixture(t, ds)
	x64ID, arm64ID, _ := f.addBoth(t, ds)

	queued := func(host *fleet.Host) []uint {
		_, err := ds.EnqueueSetupExperienceItems(ctx, "windows", "windows", host.UUID, f.team.ID)
		require.NoError(t, err)
		var ids []uint
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &ids,
				`SELECT software_installer_id FROM setup_experience_status_results WHERE host_uuid = ? AND software_installer_id IS NOT NULL`, host.UUID)
		})
		return ids
	}
	setForSetup := func(x64, arm64 bool) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE software_installers SET install_during_setup = (id = ? AND ?) OR (id = ? AND ?) WHERE id IN (?, ?)`,
				x64ID, x64, arm64ID, arm64, x64ID, arm64ID)
			return err
		})
	}

	armHost := newWindowsHostWithCPU(t, ds, f.team.ID, "ARM")
	x64Host := newWindowsHostWithCPU(t, ds, f.team.ID, "x86_64")
	unknownHost := newWindowsHostWithCPU(t, ds, f.team.ID, "")

	// Both builds selected: each host gets one, for its CPU.
	setForSetup(true, true)
	require.Equal(t, []uint{arm64ID}, queued(armHost))
	require.Equal(t, []uint{x64ID}, queued(x64Host))
	require.Equal(t, []uint{x64ID}, queued(unknownHost))

	// Only the ARM64 build selected: hosts that aren't ARM get nothing.
	setForSetup(false, true)
	require.Equal(t, []uint{arm64ID}, queued(armHost))
	require.Empty(t, queued(x64Host))

	// Only the x64 build selected: every host gets it.
	setForSetup(true, false)
	require.Equal(t, []uint{x64ID}, queued(armHost))
	require.Equal(t, []uint{x64ID}, queued(x64Host))

	// Retrying after a failure doesn't queue the other build of a title that already succeeded.
	setForSetup(true, true)
	require.Equal(t, []uint{arm64ID}, queued(armHost))
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE setup_experience_status_results SET status = 'success' WHERE host_uuid = ?`, armHost.UUID)
		return err
	})
	_, err := ds.ResetSetupExperienceItemsAfterFailure(ctx, "windows", "windows", armHost.UUID, f.team.ID)
	require.NoError(t, err)
	var rows []struct {
		InstallerID uint   `db:"software_installer_id"`
		Status      string `db:"status"`
	}
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &rows,
			`SELECT software_installer_id, status FROM setup_experience_status_results WHERE host_uuid = ? AND software_installer_id IS NOT NULL`, armHost.UUID)
	})
	require.Len(t, rows, 1)
	require.Equal(t, arm64ID, rows[0].InstallerID)
	require.Equal(t, "success", rows[0].Status)
}

func testArchSiblingsBatchSet(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newArchSiblingsFixture(t, ds)

	apply := func(payloads ...*fleet.UploadSoftwareInstallerPayload) {
		_, err := ds.BatchSetSoftwareInstallers(ctx, &f.team.ID, payloads)
		require.NoError(t, err)
	}
	titleID := func() uint {
		var id uint
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, `SELECT id FROM software_titles WHERE upgrade_code = ?`, f.upgradeCode)
		})
		return id
	}
	activeByApp := func() map[uint]uint {
		var rows []struct {
			ID    uint `db:"id"`
			FMAID uint `db:"fleet_maintained_app_id"`
		}
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &rows,
				`SELECT id, fleet_maintained_app_id FROM software_installers WHERE global_or_team_id = ? AND is_active = 1 AND fleet_maintained_app_id IS NOT NULL`, f.team.ID)
		})
		m := make(map[uint]uint, len(rows))
		for _, r := range rows {
			m[r.FMAID] = r.ID
		}
		return m
	}

	apply(f.installerArgs(f.x64, "chrome-x64-1", "1.0"), f.installerArgs(f.arm64, "chrome-arm64-1", "1.0"))
	active := activeByApp()
	require.Len(t, active, 2)
	require.Equal(t, []uint{active[f.x64.ID], active[f.arm64.ID]}, activeInstallerIDs(t, ds, f.team.ID, titleID()))

	// Re-applying the same config changes nothing.
	apply(f.installerArgs(f.x64, "chrome-x64-1", "1.0"), f.installerArgs(f.arm64, "chrome-arm64-1", "1.0"))
	require.Equal(t, active, activeByApp())

	// A pin on one build doesn't touch the other's.
	x64v2 := f.installerArgs(f.x64, "chrome-x64-2", "2.0")
	arm64Pinned := f.installerArgs(f.arm64, "chrome-arm64-2", "2.0")
	arm64Pinned.RollbackVersion = "1.0"
	apply(x64v2, arm64Pinned)
	newActive := activeByApp()
	require.NotEqual(t, active[f.x64.ID], newActive[f.x64.ID])
	require.Equal(t, active[f.arm64.ID], newActive[f.arm64.ID])
	pin, err := ds.GetPinnedVersion(ctx, &f.team.ID, titleID(), f.arm64.ID)
	require.NoError(t, err)
	require.Equal(t, "1.0", *pin)
	_, err = ds.GetPinnedVersion(ctx, &f.team.ID, titleID(), f.x64.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// A policy on the ARM64 build moves to the x64 build when the ARM64 build is dropped from the config.
	policy, err := ds.NewTeamPolicy(ctx, f.team.ID, nil, fleet.PolicyPayload{
		Name: "Chrome ARM64", Query: "SELECT 1;", Platform: "windows", SoftwareInstallerID: new(newActive[f.arm64.ID]),
	})
	require.NoError(t, err)
	apply(f.installerArgs(f.x64, "chrome-x64-2", "2.0"))
	require.Equal(t, map[uint]uint{f.x64.ID: newActive[f.x64.ID]}, activeByApp())
	var arm64Rows int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &arm64Rows, `SELECT COUNT(*) FROM software_installers WHERE fleet_maintained_app_id = ?`, f.arm64.ID)
	})
	require.Zero(t, arm64Rows)
	policy, err = ds.Policy(ctx, policy.ID)
	require.NoError(t, err)
	require.Equal(t, newActive[f.x64.ID], *policy.SoftwareInstallerID)
	_, err = ds.GetPinnedVersion(ctx, &f.team.ID, titleID(), f.arm64.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// An unrelated FMA can't join the title.
	other, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name: "Chromium", Slug: "chromium/windows", Platform: "windows", UniqueIdentifier: "Google Chrome",
	})
	require.NoError(t, err)
	_, err = ds.BatchSetSoftwareInstallers(ctx, &f.team.ID, []*fleet.UploadSoftwareInstallerPayload{
		f.installerArgs(f.x64, "chrome-x64-2", "2.0"), f.installerArgs(other, "chromium-1", "1.0"),
	})
	require.ErrorContains(t, err, "Only one of Google Chrome or Chromium can be added to the same fleet")
}
