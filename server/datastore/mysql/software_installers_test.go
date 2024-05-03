package mysql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/test"
 	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestSoftwareInstallers(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"testListSoftwareInstallerDetails", testListSoftwareInstallerDetails},
		{"InsertSoftwareInstallRequest", testInsertSoftwareInstallRequest},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testListSoftwareInstallerDetails(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "host2key", "host2uuid", time.Now())

	script1, err := insertScriptContents(ctx, "hello", ds.writer(ctx))
	require.NoError(t, err)
	script1Id, err := script1.LastInsertId()
	require.NoError(t, err)

	script2, err := insertScriptContents(ctx, "world", ds.writer(ctx))
	require.NoError(t, err)
	script2Id, err := script2.LastInsertId()
	require.NoError(t, err)

	installer1, err := insertSoftwareInstaller(ctx, ds.writer(ctx), "file1", "1.0", "SELECT 1", "storage1", script1Id, script2Id)
	require.NoError(t, err)
	installer1Id, err := installer1.LastInsertId()
	require.NoError(t, err)

	installer2, err := insertSoftwareInstaller(ctx, ds.writer(ctx), "file2", "2.0", "SELECT 2", "storage2", script2Id, script1Id)
	require.NoError(t, err)
	installer2Id, err := installer2.LastInsertId()
	require.NoError(t, err)

	hostInstall1, err := insertHostSoftwareInstalls(ctx, ds.writer(ctx), host1.ID, "exec1", uint(installer1Id))
	require.NoError(t, err)
	_ = hostInstall1

	hostInstall2, err := insertHostSoftwareInstalls(ctx, ds.writer(ctx), host1.ID, "exec2", uint(installer2Id))
	require.NoError(t, err)
	_ = hostInstall2

	hostInstall3, err := insertHostSoftwareInstalls(ctx, ds.writer(ctx), host2.ID, "exec3", uint(installer1Id))
	require.NoError(t, err)
	_ = hostInstall3

	hostInstall4, err := insertHostSoftwareInstalls(ctx, ds.writer(ctx), host2.ID, "exec4", uint(installer2Id))
	require.NoError(t, err)
	hostInstall4Id, err := hostInstall4.LastInsertId()
	require.NoError(t, err)

	_ = ds.writer(ctx).MustExec("UPDATE host_software_installs SET install_script_exit_code = 0 WHERE id = ?", hostInstall4Id)

	installDetailsList1, err := ds.ListPendingSoftwareInstallDetails(ctx, host1.ID)
	require.NoError(t, err)
	require.Equal(t, 2, len(installDetailsList1))

	installDetailsList2, err := ds.ListPendingSoftwareInstallDetails(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(installDetailsList2))

	require.Equal(t, host1.ID, installDetailsList1[0].HostID)
	require.Equal(t, host1.ID, installDetailsList1[1].HostID)
	require.Equal(t, "exec1", installDetailsList1[0].ExecutionID)
	require.Equal(t, "exec2", installDetailsList1[1].ExecutionID)
	require.Equal(t, []byte("hello"), installDetailsList1[0].InstallScript)
	require.Equal(t, []byte("world"), installDetailsList1[1].InstallScript)
	require.Equal(t, []byte("world"), installDetailsList1[0].PostInstallScript)
	require.Equal(t, []byte("hello"), installDetailsList1[1].PostInstallScript)
	require.Equal(t, uint(installer1Id), installDetailsList1[0].InstallerID)
	require.Equal(t, uint(installer2Id), installDetailsList1[1].InstallerID)
	require.Equal(t, "SELECT 1", installDetailsList1[0].PreInstallCondition)
	require.Equal(t, "SELECT 2", installDetailsList1[1].PreInstallCondition)

	require.Equal(t, installDetailsList1[0].InstallerID, installDetailsList2[0].InstallerID)

	exec1, err := ds.GetSoftwareInstallDetails(ctx, "exec1")
	require.NoError(t, err)

	require.Equal(t, host1.ID, exec1.HostID)
	require.Equal(t, "exec1", exec1.ExecutionID)
	require.Equal(t, []byte("hello"), exec1.InstallScript)
	require.Equal(t, []byte("world"), exec1.PostInstallScript)
	require.Equal(t, uint(installer1Id), exec1.InstallerID)
	require.Equal(t, "SELECT 1", exec1.PreInstallCondition)
}

func insertHostSoftwareInstalls(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostId uint,
	executionId string,
	softwareInstallerId uint,
) (sql.Result, error) {
	stmt := `
  INSERT INTO host_software_installs (
    host_id,
    execution_id,
    software_installer_id
  ) VALUES (?, ?, ?)
`
	res, err := tx.ExecContext(ctx, stmt, hostId, executionId, softwareInstallerId)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting host software install")
	}

	return res, nil
}

func insertSoftwareInstaller(
	ctx context.Context,
	tx sqlx.ExtContext,
	filename,
	version,
	preinstallQuery,
	storageId string,
	installScriptId,
	postInstallScriptId int64,
) (sql.Result, error) {
	stmt := `
  INSERT INTO software_installers (
    filename,
    version,
    pre_install_query,
    install_script_content_id,
    post_install_script_content_id,
    storage_id
  )
  VALUES (?, ?, ?, ?, ?, ?)
`
	res, err := tx.ExecContext(ctx,
		stmt,
		filename,
		version,
		preinstallQuery,
		installScriptId,
		postInstallScriptId,
		storageId,
	)

	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting software installer")
	}

	return res, nil
}

func testInsertSoftwareInstallRequest(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	cases := map[string]*uint{
		"no team": nil,
		"team":    &team.ID,
	}

	for tc, teamID := range cases {
		t.Run(tc, func(t *testing.T) {
			// non-existent installer and host does the installer check first
			err := ds.InsertSoftwareInstallRequest(ctx, 1, 1, teamID)
			var nfe fleet.NotFoundError
			require.ErrorAs(t, err, &nfe)

			// non-existent host
			installerID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				Title:         "foo",
				Source:        "bar",
				InstallScript: "echo",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			installerMeta, err := ds.GetSoftwareInstallerMetadata(ctx, installerID)
			require.NoError(t, err)

			err = ds.InsertSoftwareInstallRequest(ctx, 12, installerMeta.TitleID, teamID)
			require.ErrorAs(t, err, &nfe)

			// successful insert
			host, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "macos-test" + tc,
				OsqueryHostID: ptr.String("osquery-macos" + tc),
				NodeKey:       ptr.String("node-key-macos" + tc),
				UUID:          uuid.NewString(),
				Platform:      "darwin",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			err = ds.InsertSoftwareInstallRequest(ctx, host.ID, installerMeta.TitleID, teamID)
			require.NoError(t, err)
		})
	}
}
