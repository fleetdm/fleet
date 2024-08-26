package mysql

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/filesystem"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
		{"HasSelfServiceSoftwareInstallers", testHasSelfServiceSoftwareInstallers},
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

	installerID1, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		InstallerFile:     bytes.NewReader([]byte("hello")),
		StorageID:         "storage1",
		Filename:          "file1",
		Title:             "file1",
		Version:           "1.0",
		Source:            "apps",
	})
	require.NoError(t, err)

	installerID2, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "world",
		PreInstallQuery:   "SELECT 2",
		PostInstallScript: "hello",
		InstallerFile:     bytes.NewReader([]byte("hello")),
		StorageID:         "storage2",
		Filename:          "file2",
		Title:             "file2",
		Version:           "2.0",
		Source:            "apps",
	})
	require.NoError(t, err)

	installerID3, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "banana",
		PreInstallQuery:   "SELECT 3",
		PostInstallScript: "apple",
		InstallerFile:     bytes.NewReader([]byte("hello")),
		StorageID:         "storage3",
		Filename:          "file3",
		Title:             "file3",
		Version:           "3.0",
		Source:            "apps",
		SelfService:       true,
	})
	require.NoError(t, err)

	hostInstall1, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID1, false)
	require.NoError(t, err)

	hostInstall2, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID2, false)
	require.NoError(t, err)

	hostInstall3, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, installerID1, false)
	require.NoError(t, err)

	hostInstall4, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, installerID2, false)
	require.NoError(t, err)

	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host2.ID,
		InstallUUID:           hostInstall4,
		InstallScriptExitCode: ptr.Int(0),
	})
	require.NoError(t, err)

	hostInstall5, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, installerID2, false)
	require.NoError(t, err)

	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                    host2.ID,
		InstallUUID:               hostInstall5,
		PreInstallConditionOutput: ptr.String("output"),
	})
	require.NoError(t, err)

	installDetailsList1, err := ds.ListPendingSoftwareInstalls(ctx, host1.ID)
	require.NoError(t, err)
	require.Equal(t, 2, len(installDetailsList1))

	installDetailsList2, err := ds.ListPendingSoftwareInstalls(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(installDetailsList2))

	require.Contains(t, installDetailsList1, hostInstall1)
	require.Contains(t, installDetailsList1, hostInstall2)

	require.Contains(t, installDetailsList2, hostInstall3)

	exec1, err := ds.GetSoftwareInstallDetails(ctx, hostInstall1)
	require.NoError(t, err)

	require.Equal(t, host1.ID, exec1.HostID)
	require.Equal(t, hostInstall1, exec1.ExecutionID)
	require.Equal(t, "hello", exec1.InstallScript)
	require.Equal(t, "world", exec1.PostInstallScript)
	require.Equal(t, installerID1, exec1.InstallerID)
	require.Equal(t, "SELECT 1", exec1.PreInstallCondition)
	require.False(t, exec1.SelfService)

	hostInstall6, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID3, true)
	require.NoError(t, err)

	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                    host1.ID,
		InstallUUID:               hostInstall6,
		PreInstallConditionOutput: ptr.String("output"),
	})
	require.NoError(t, err)

	exec2, err := ds.GetSoftwareInstallDetails(ctx, hostInstall6)
	require.NoError(t, err)

	require.Equal(t, host1.ID, exec2.HostID)
	require.Equal(t, hostInstall6, exec2.ExecutionID)
	require.Equal(t, "banana", exec2.InstallScript)
	require.Equal(t, "apple", exec2.PostInstallScript)
	require.Equal(t, installerID3, exec2.InstallerID)
	require.Equal(t, "SELECT 3", exec2.PreInstallCondition)
	require.True(t, exec2.SelfService)
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
			si, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, teamID, 1, false)
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
			installerMeta, err := ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
			require.NoError(t, err)

			si, err = ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, teamID, *installerMeta.TitleID, false)
			require.NoError(t, err)
			require.NotNil(t, si)
			require.Equal(t, "foo.pkg", si.Name)

			// non-existent host
			_, err = ds.InsertSoftwareInstallRequest(ctx, 12, si.InstallerID, false)
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
			_, err = ds.InsertSoftwareInstallRequest(ctx, host.ID, si.InstallerID, false)
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
		expectedStatus          fleet.SoftwareInstallerStatus
		postInstallScriptEC     *int
		preInstallQueryOutput   *string
		installScriptEC         *int
		postInstallScriptOutput *string
		installScriptOutput     *string
	}{
		{
			name:                    "pending install",
			expectedStatus:          fleet.SoftwareInstallerPending,
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
		{
			name:                    "failing install post install script",
			expectedStatus:          fleet.SoftwareInstallerFailed,
			postInstallScriptEC:     ptr.Int(1),
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
		{
			name:                    "failing install install script",
			expectedStatus:          fleet.SoftwareInstallerFailed,
			installScriptEC:         ptr.Int(1),
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
		{
			name:                    "failing install pre install query",
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

			installUUID, err := ds.InsertSoftwareInstallRequest(ctx, host.ID, installerID, false)
			require.NoError(t, err)
			err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
				HostID:                    host.ID,
				InstallUUID:               installUUID,
				PreInstallConditionOutput: tc.preInstallQueryOutput,
				InstallScriptExitCode:     tc.installScriptEC,
				InstallScriptOutput:       tc.installScriptOutput,
				PostInstallScriptExitCode: tc.postInstallScriptEC,
				PostInstallScriptOutput:   tc.postInstallScriptOutput,
			})
			require.NoError(t, err)

			res, err := ds.GetSoftwareInstallResults(ctx, installUUID)
			require.NoError(t, err)

			require.Equal(t, installUUID, res.InstallUUID)
			require.Equal(t, tc.expectedStatus, res.Status)
			require.Equal(t, swFilename, res.SoftwarePackage)
			require.Equal(t, host.ID, res.HostID)
			require.Equal(t, tc.preInstallQueryOutput, res.PreInstallQueryOutput)
			require.Equal(t, tc.postInstallScriptOutput, res.PostInstallScriptOutput)
			require.Equal(t, tc.installScriptOutput, res.Output)
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
	err = ds.CleanupUnusedSoftwareInstallers(ctx, store, time.Now())
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
	err = ds.CleanupUnusedSoftwareInstallers(ctx, store, time.Now())
	require.NoError(t, err)
	assertExisting([]string{ins0})

	// remove it from the DB, will now cleanup
	err = ds.DeleteSoftwareInstaller(ctx, swi)
	require.NoError(t, err)

	// would clean up, but not created before 1m ago
	err = ds.CleanupUnusedSoftwareInstallers(ctx, store, time.Now().Add(-time.Minute))
	require.NoError(t, err)
	assertExisting([]string{ins0})

	// do actual cleanup
	err = ds.CleanupUnusedSoftwareInstallers(ctx, store, time.Now().Add(time.Minute))
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
			meta, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, title.ID, false)
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
	installerMeta, err := ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
	require.NoError(t, err)

	metaByTeamAndTitle, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, *installerMeta.TitleID, true)
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
	installerMeta, err = ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
	require.NoError(t, err)

	metaByTeamAndTitle, err = ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, *installerMeta.TitleID, true)
	require.NoError(t, err)
	require.Equal(t, "echo install", metaByTeamAndTitle.InstallScript)
	require.Equal(t, "", metaByTeamAndTitle.PostInstallScript)
	require.EqualValues(t, installerID, metaByTeamAndTitle.InstallerID)
	require.Equal(t, "", metaByTeamAndTitle.PreInstallQuery)
}

func testHasSelfServiceSoftwareInstallers(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	const platform = "linux"
	// No installers
	hasSelfService, err := ds.HasSelfServiceSoftwareInstallers(ctx, platform, nil)
	require.NoError(t, err)
	assert.False(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, &team.ID)
	require.NoError(t, err)
	assert.False(t, hasSelfService)

	// Create a non-self service installer
	_, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:         "foo",
		Source:        "bar",
		InstallScript: "echo install",
		TeamID:        &team.ID,
		Filename:      "foo.pkg",
		Platform:      platform,
		SelfService:   false,
	})
	require.NoError(t, err)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, nil)
	require.NoError(t, err)
	assert.False(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, &team.ID)
	require.NoError(t, err)
	assert.False(t, hasSelfService)

	// Create a self-service installer for team
	_, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:         "foo2",
		Source:        "bar2",
		InstallScript: "echo install",
		TeamID:        &team.ID,
		Filename:      "foo2.pkg",
		Platform:      platform,
		SelfService:   true,
	})
	require.NoError(t, err)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, nil)
	require.NoError(t, err)
	assert.False(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, &team.ID)
	require.NoError(t, err)
	assert.True(t, hasSelfService)

	// Create a non self-service VPP for global/linux (not truly possible as VPP is Apple but for testing)
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_1", Platform: platform}}, Name: "vpp1", BundleIdentifier: "com.app.vpp1"}, nil)
	require.NoError(t, err)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, nil)
	require.NoError(t, err)
	assert.False(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, &team.ID)
	require.NoError(t, err)
	assert.True(t, hasSelfService)

	// Create a self-service VPP for global/linux (not truly possible as VPP is Apple but for testing)
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_2", Platform: platform}, SelfService: true}, Name: "vpp2", BundleIdentifier: "com.app.vpp2"}, nil)
	require.NoError(t, err)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, nil)
	require.NoError(t, err)
	assert.True(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, &team.ID)
	require.NoError(t, err)
	assert.True(t, hasSelfService)

	// Create a global self-service installer
	_, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:         "foo global",
		Source:        "bar",
		InstallScript: "echo install",
		TeamID:        nil,
		Filename:      "foo global.pkg",
		Platform:      platform,
		SelfService:   true,
	})
	require.NoError(t, err)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, "ubuntu", nil)
	require.NoError(t, err)
	assert.True(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, "ubuntu", &team.ID)
	require.NoError(t, err)
	assert.True(t, hasSelfService)

	// Create a self-service VPP for team/darwin
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_3", Platform: fleet.MacOSPlatform}, SelfService: true}, Name: "vpp3", BundleIdentifier: "com.app.vpp3"}, &team.ID)
	require.NoError(t, err)
	// Check darwin
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, "darwin", nil)
	require.NoError(t, err)
	assert.False(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, "darwin", &team.ID)
	require.NoError(t, err)
	assert.True(t, hasSelfService)
}
