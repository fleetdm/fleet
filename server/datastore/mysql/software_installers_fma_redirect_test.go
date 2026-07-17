package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// TestFMAActiveInstallerRedirect verifies that flipping the active installer for a
// title (auto-update promotion / pin change) redirects installs frozen on the
// superseded version to the new one, and that ResolveActiveInstallerForFrozen
// reports the current active sibling for a retry.
func TestFMAActiveInstallerRedirect(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "fma-redirect"})
	require.NoError(t, err)
	teamID := team.ID
	user := test.NewUser(t, ds, "Admin", "admin-fma@example.com", true)

	// The initially-active installer and its title.
	oldID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:           "Redirector",
		Source:          "apps",
		InstallScript:   "echo old",
		Version:         "1.0.0",
		TeamID:          &teamID,
		Filename:        "redirector-1.0.0.pkg",
		StorageID:       "storage-old",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// A newer cached version for the same title, cloned from the old row and left
	// inactive, then make the old row the active one to start.
	var newID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx, `
			INSERT INTO software_installers
				(team_id, global_or_team_id, title_id, filename, version, platform, pre_install_query,
				 install_script_content_id, post_install_script_content_id, storage_id, self_service,
				 user_id, user_name, user_email, url, package_ids, extension, uninstall_script_content_id,
				 fleet_maintained_app_id, install_during_setup, upgrade_code, is_active, patch_query, http_etag)
			SELECT team_id, global_or_team_id, title_id, 'redirector-2.0.0.pkg', '2.0.0', platform, pre_install_query,
				 install_script_content_id, post_install_script_content_id, 'storage-new', self_service,
				 user_id, user_name, user_email, url, package_ids, extension, uninstall_script_content_id,
				 fleet_maintained_app_id, install_during_setup, upgrade_code, 0, patch_query, http_etag
			FROM software_installers WHERE id = ?`, oldID)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		newID = uint(id) //nolint:gosec
		_, err = q.ExecContext(ctx,
			`UPDATE software_installers SET is_active = (id = ?) WHERE title_id = ? AND global_or_team_id = ?`,
			oldID, titleID, teamID)
		return err
	})

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "fma-host",
		OsqueryHostID: new("fma-osq"),
		NodeKey:       new("fma-nk"),
		UUID:          uuid.NewString(),
		Platform:      "darwin",
		TeamID:        &teamID,
	})
	require.NoError(t, err)

	// Before promotion: the inactive new row resolves to the active old row; the
	// active old row resolves to itself; an unknown id is returned unchanged.
	requireResolves := func(frozen, want uint) {
		t.Helper()
		got, err := ds.ResolveActiveInstallerForFrozen(ctx, frozen)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	requireResolves(newID, oldID)
	requireResolves(oldID, oldID)
	requireResolves(999999, 999999)

	// Queue two installs on the old (active) installer: the first activates and is
	// dispatched, the second stays queued.
	activatedUUID, err := ds.InsertSoftwareInstallRequest(ctx, host.ID, oldID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	queuedUUID, err := ds.InsertSoftwareInstallRequest(ctx, host.ID, oldID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	// Promote: flip the active installer to the new version.
	require.NoError(t, ds.SetFleetMaintainedAppActiveInstaller(ctx,
		&fleet.UpdateSoftwareInstallerPayload{TeamID: &teamID, TitleID: titleID}, newID))

	// The active flip took effect, so retries now resolve to the new version.
	requireResolves(oldID, newID)

	// The still-queued install was redirected to the new version rather than
	// dropped, so it installs the version Fleet now displays.
	var queued struct {
		InstallerID uint   `db:"software_installer_id"`
		Version     string `db:"version"`
		Filename    string `db:"installer_filename"`
		Canceled    bool   `db:"canceled"`
	}
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &queued,
			`SELECT software_installer_id, version, installer_filename, canceled FROM host_software_installs WHERE execution_id = ?`,
			queuedUUID)
	})
	require.Equal(t, newID, queued.InstallerID, "queued install redirected to the active installer")
	require.Equal(t, "2.0.0", queued.Version)
	require.Equal(t, "redirector-2.0.0.pkg", queued.Filename)
	require.False(t, queued.Canceled)

	// The already-dispatched install (can't be recalled from the host) was canceled;
	// its automation re-queues against the active installer.
	var dispatchedCanceled bool
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &dispatchedCanceled,
			`SELECT canceled FROM host_software_installs WHERE execution_id = ?`, activatedUUID)
	})
	require.True(t, dispatchedCanceled, "dispatched install on the superseded version was canceled")
}
