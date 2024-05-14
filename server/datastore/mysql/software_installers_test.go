package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/filesystem"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
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
		{"SoftwareInstallRequests", testSoftwareInstallRequests},
		{"ListPendingSoftwareInstalls", testListPendingSoftwareInstalls},
		{"GetSoftwareInstallResults", testGetSoftwareInstallResult},
		{"CleanupUnusedSoftwareInstallers", testCleanupUnusedSoftwareInstallers},
		{"BatchSetSoftwareInstallers", testBatchSetSoftwareInstallers},
		{"GetSoftwareInstallerMetadataByTeamAndTitleID", testGetSoftwareInstallerMetadataByTeamAndTitleID},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testListPendingSoftwareInstalls(t *testing.T, ds *Datastore) {
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

	hostInstall5, err := insertHostSoftwareInstalls(ctx, ds.writer(ctx), host2.ID, "exec5", uint(installer2Id))
	require.NoError(t, err)
	hostInstall5Id, err := hostInstall5.LastInsertId()
	require.NoError(t, err)

	_ = ds.writer(ctx).MustExec("UPDATE host_software_installs SET pre_install_query_output = 'output' WHERE id = ?", hostInstall5Id)

	installDetailsList1, err := ds.ListPendingSoftwareInstalls(ctx, host1.ID)
	require.NoError(t, err)
	require.Equal(t, 2, len(installDetailsList1))

	installDetailsList2, err := ds.ListPendingSoftwareInstalls(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(installDetailsList2))

	require.Contains(t, installDetailsList1, "exec1")
	require.Contains(t, installDetailsList1, "exec2")

	require.Contains(t, installDetailsList2, "exec3")

	exec1, err := ds.GetSoftwareInstallDetails(ctx, "exec1")
	require.NoError(t, err)

	require.Equal(t, host1.ID, exec1.HostID)
	require.Equal(t, "exec1", exec1.ExecutionID)
	require.Equal(t, "hello", exec1.InstallScript)
	require.Equal(t, "world", exec1.PostInstallScript)
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

func testSoftwareInstallRequests(t *testing.T, ds *Datastore) {
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
			// non-existent installer
			si, err := ds.GetSoftwareInstallerForTitle(ctx, 1, teamID)
			var nfe fleet.NotFoundError
			require.ErrorAs(t, err, &nfe)
			require.Nil(t, si)

			installerID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				Title:         "foo",
				Source:        "bar",
				InstallScript: "echo",
				TeamID:        teamID,
				Filename:      "foo.pkg",
			})
			require.NoError(t, err)
			installerMeta, err := ds.GetSoftwareInstallerMetadata(ctx, installerID)
			require.NoError(t, err)

			si, err = ds.GetSoftwareInstallerForTitle(ctx, *installerMeta.TitleID, teamID)
			require.NoError(t, err)
			require.NotNil(t, si)
			require.Equal(t, "foo.pkg", si.Name)

			// non-existent host
			err = ds.InsertSoftwareInstallRequest(ctx, 12, si.InstallerID)
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
			err = ds.InsertSoftwareInstallRequest(ctx, host.ID, si.InstallerID)
			require.NoError(t, err)

			// list hosts with software install requests
			userTeamFilter := fleet.TeamFilter{
				User: &fleet.User{GlobalRole: ptr.String("admin")},
			}
			expectStatus := fleet.SoftwareInstallerPending
			hosts, err := ds.ListHosts(ctx, userTeamFilter, fleet.HostListOptions{
				ListOptions:           fleet.ListOptions{PerPage: 100},
				SoftwareTitleIDFilter: installerMeta.TitleID,
				SoftwareStatusFilter:  &expectStatus,
				TeamFilter:            teamID,
			})
			require.NoError(t, err)
			require.Len(t, hosts, 1)
			require.Equal(t, host.ID, hosts[0].ID)

			// get software title includes status
			summary, err := ds.GetSummaryHostSoftwareInstalls(ctx, installerMeta.InstallerID)
			require.NoError(t, err)
			require.Equal(t, fleet.SoftwareInstallerStatusSummary{
				Installed: 0,
				Pending:   1,
				Failed:    0,
			}, *summary)
		})
	}
}

func testGetSoftwareInstallResult(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)
	teamID := team.ID

	for _, tc := range []struct {
		name                    string
		uuid                    string
		expectedStatus          fleet.SoftwareInstallerStatus
		postInstallScriptEC     *uint
		preInstallQueryOutput   *string
		installScriptEC         *uint
		postInstallScriptOutput *string
		installScriptOutput     *string
	}{
		{
			name:                    "pending install",
			uuid:                    "pending",
			expectedStatus:          fleet.SoftwareInstallerPending,
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
		{
			name:                    "failing install post install script",
			uuid:                    "fail_post_install_script",
			expectedStatus:          fleet.SoftwareInstallerFailed,
			postInstallScriptEC:     ptr.Uint(1),
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
		{
			name:                    "failing install install script",
			uuid:                    "fail_install_script",
			expectedStatus:          fleet.SoftwareInstallerFailed,
			installScriptEC:         ptr.Uint(1),
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
		{
			name:                    "failing install pre install query",
			uuid:                    "fail_pre_install_query",
			expectedStatus:          fleet.SoftwareInstallerFailed,
			preInstallQueryOutput:   ptr.String(""),
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// create a host and software installer
			swFilename := "file_" + tc.name + ".pkg"
			installerID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				Title:         "foo" + tc.name,
				Source:        "bar" + tc.name,
				InstallScript: "echo " + tc.name,
				TeamID:        &teamID,
				Filename:      swFilename,
			})
			require.NoError(t, err)
			host, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "macos-test-" + tc.name,
				ComputerName:  "macos-test-" + tc.name,
				OsqueryHostID: ptr.String("osquery-macos-" + tc.name),
				NodeKey:       ptr.String("node-key-macos-" + tc.name),
				UUID:          uuid.NewString(),
				Platform:      "darwin",
				TeamID:        &teamID,
			})
			require.NoError(t, err)

			// Need to insert manually so we have access to the UUID (it's generated in the DS method)
			query := `INSERT INTO host_software_installs (execution_id, host_id, software_installer_id, post_install_script_exit_code, install_script_exit_code, pre_install_query_output, install_script_output, post_install_script_output) VALUES (?,?,?,?,?,?,?,?)`
			_, err = ds.writer(ctx).ExecContext(ctx, query, tc.uuid, host.ID, installerID, tc.postInstallScriptEC, tc.installScriptEC, tc.preInstallQueryOutput, tc.installScriptOutput, tc.postInstallScriptOutput)
			require.NoError(t, err)

			res, err := ds.GetSoftwareInstallResults(ctx, tc.uuid)
			require.NoError(t, err)

			require.Equal(t, tc.uuid, res.InstallUUID)
			require.Equal(t, tc.expectedStatus, res.Status)
			require.Equal(t, swFilename, res.SoftwarePackage)
			require.Equal(t, host.ID, res.HostID)
			require.Equal(t, host.DisplayName(), res.HostDisplayName)
			expectedPreInstallQueryOutput := ""
			if tc.preInstallQueryOutput != nil {
				expectedPreInstallQueryOutput = *tc.preInstallQueryOutput
			}
			require.Equal(t, expectedPreInstallQueryOutput, res.PreInstallQueryOutput)

			expectedPostInstallScriptOutput := ""
			if tc.postInstallScriptOutput != nil {
				expectedPostInstallScriptOutput = *tc.postInstallScriptOutput
			}
			require.Equal(t, expectedPostInstallScriptOutput, res.PostInstallScriptOutput)
			expectedInstallScriptOutput := ""
			if tc.installScriptOutput != nil {
				expectedInstallScriptOutput = *tc.installScriptOutput
			}
			require.Equal(t, expectedInstallScriptOutput, res.Output)
		})
	}
}

func testCleanupUnusedSoftwareInstallers(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	dir := t.TempDir()
	store, err := filesystem.NewSoftwareInstallerStore(dir)
	require.NoError(t, err)

	assertExisting := func(want []string) {
		dirEnts, err := os.ReadDir(filepath.Join(dir, "software-installers"))
		require.NoError(t, err)
		got := make([]string, 0, len(dirEnts))
		for _, de := range dirEnts {
			if de.Type().IsRegular() {
				got = append(got, de.Name())
			}
		}
		require.ElementsMatch(t, want, got)
	}

	// cleanup an empty store
	err = ds.CleanupUnusedSoftwareInstallers(ctx, store)
	require.NoError(t, err)
	assertExisting(nil)

	// put an installer and save it in the DB
	ins0 := "installer0"
	ins0File := bytes.NewReader([]byte("installer0"))
	err = store.Put(ctx, ins0, ins0File)
	require.NoError(t, err)
	assertExisting([]string{ins0})

	swi, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		InstallerFile: ins0File,
		StorageID:     ins0,
		Filename:      "installer0",
		Title:         "ins0",
		Source:        "apps",
	})
	require.NoError(t, err)

	assertExisting([]string{ins0})
	err = ds.CleanupUnusedSoftwareInstallers(ctx, store)
	require.NoError(t, err)
	assertExisting([]string{ins0})

	// remove it from the DB, will now cleanup
	err = ds.DeleteSoftwareInstaller(ctx, swi)
	require.NoError(t, err)

	err = ds.CleanupUnusedSoftwareInstallers(ctx, store)
	require.NoError(t, err)
	assertExisting(nil)
}

func testBatchSetSoftwareInstallers(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	// TODO(roberto): perform better assertions, we should have evertything
	// to check that the actual values of everything match.
	assertSoftware := func(wantTitles []fleet.SoftwareTitle) {
		tmFilter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
		titles, _, _, err := ds.ListSoftwareTitles(
			ctx,
			fleet.SoftwareTitleListOptions{TeamID: &team.ID},
			tmFilter,
		)
		require.NoError(t, err)
		require.Len(t, titles, len(wantTitles))

		for _, title := range titles {
			meta, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, title.ID)
			require.NoError(t, err)
			require.NotNil(t, meta.TitleID)
		}
	}

	// batch set with everything empty
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, nil)
	require.NoError(t, err)
	assertSoftware(nil)
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{})
	require.NoError(t, err)
	assertSoftware(nil)

	// add a single installer
	ins0 := "installer0"
	ins0File := bytes.NewReader([]byte("installer0"))
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{{
		InstallScript:   "install",
		InstallerFile:   ins0File,
		StorageID:       ins0,
		Filename:        "installer0",
		Title:           "ins0",
		Source:          "apps",
		Version:         "1",
		PreInstallQuery: "foo",
	}})
	require.NoError(t, err)
	assertSoftware([]fleet.SoftwareTitle{
		{Name: ins0, Source: "apps", Browser: ""},
	})

	// add a new installer + ins0 installer
	ins1 := "installer1"
	ins1File := bytes.NewReader([]byte("installer1"))
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:   "install",
			InstallerFile:   ins0File,
			StorageID:       ins0,
			Filename:        ins0,
			Title:           ins0,
			Source:          "apps",
			Version:         "1",
			PreInstallQuery: "select 0 from foo;",
		},
		{
			InstallScript:     "install",
			PostInstallScript: "post-install",
			InstallerFile:     ins1File,
			StorageID:         ins1,
			Filename:          ins1,
			Title:             ins1,
			Source:            "apps",
			Version:           "2",
			PreInstallQuery:   "select 1 from bar;",
		},
	})
	require.NoError(t, err)
	assertSoftware([]fleet.SoftwareTitle{
		{Name: ins0, Source: "apps", Browser: ""},
		{Name: ins1, Source: "apps", Browser: ""},
	})

	// remove ins0
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:     "install",
			PostInstallScript: "post-install",
			InstallerFile:     ins1File,
			StorageID:         ins1,
			Filename:          ins1,
			Title:             ins1,
			Source:            "apps",
			Version:           "2",
			PreInstallQuery:   "select 1 from bar;",
		},
	})
	require.NoError(t, err)
	assertSoftware([]fleet.SoftwareTitle{
		{Name: ins1, Source: "apps", Browser: ""},
	})

	// remove everything
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{})
	require.NoError(t, err)
	assertSoftware([]fleet.SoftwareTitle{})
}

func testGetSoftwareInstallerMetadataByTeamAndTitleID(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	installerID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:             "foo",
		Source:            "bar",
		InstallScript:     "echo install",
		PostInstallScript: "echo post-install",
		PreInstallQuery:   "SELECT 1",
		TeamID:            &team.ID,
		Filename:          "foo.pkg",
	})
	require.NoError(t, err)
	installerMeta, err := ds.GetSoftwareInstallerMetadata(ctx, installerID)
	require.NoError(t, err)

	metaByTeamAndTitle, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, *installerMeta.TitleID)
	require.NoError(t, err)
	require.Equal(t, "echo install", metaByTeamAndTitle.InstallScript)
	require.Equal(t, "echo post-install", metaByTeamAndTitle.PostInstallScript)
	require.EqualValues(t, installerID, metaByTeamAndTitle.InstallerID)
	require.Equal(t, "SELECT 1", metaByTeamAndTitle.PreInstallQuery)

	installerID, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:         "bar",
		Source:        "bar",
		InstallScript: "echo install",
		TeamID:        &team.ID,
		Filename:      "foo.pkg",
	})
	require.NoError(t, err)
	installerMeta, err = ds.GetSoftwareInstallerMetadata(ctx, installerID)
	require.NoError(t, err)

	metaByTeamAndTitle, err = ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, *installerMeta.TitleID)
	require.NoError(t, err)
	require.Equal(t, "echo install", metaByTeamAndTitle.InstallScript)
	require.Equal(t, "", metaByTeamAndTitle.PostInstallScript)
	require.EqualValues(t, installerID, metaByTeamAndTitle.InstallerID)
	require.Equal(t, "", metaByTeamAndTitle.PreInstallQuery)
}
