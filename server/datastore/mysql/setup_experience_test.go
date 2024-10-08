package mysql

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupExperience(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"GetSetupExperienceTitles", testGetSetupExperienceTitles},
		{"SetSetupExperienceTitles", testSetSetupExperienceTitles},
		{"ListSetupExperienceStatusResults", testSetupExperienceStatusResults},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testGetSetupExperienceTitles(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	installerID1, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		UninstallScript:   "goodbye",
		InstallerFile:     bytes.NewReader([]byte("hello")),
		StorageID:         "storage1",
		Filename:          "file1",
		Title:             "file1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
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
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.MacOSPlatform),
	})
	require.NoError(t, err)

	installerID4, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "pear",
		PreInstallQuery:   "SELECT 4",
		PostInstallScript: "apple",
		InstallerFile:     bytes.NewReader([]byte("hello2")),
		StorageID:         "storage3",
		Filename:          "file4",
		Title:             "file4",
		Version:           "4.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.IOSPlatform),
	})
	require.NoError(t, err)

	titles, count, meta, err := ds.ListSetupExperienceSoftwareTitles(ctx, team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 0)
	assert.Equal(t, 0, count)
	assert.NotNil(t, meta)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id IN (?, ?, ?)", installerID1, installerID3, installerID4)
		return err
	})

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 1)
	assert.Equal(t, installerID1, titles[0].ID)
	assert.Equal(t, 1, count)
	assert.NotNil(t, meta)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, team2.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 1)
	assert.Equal(t, installerID3, titles[0].ID)
	assert.Equal(t, 1, count)
	assert.NotNil(t, meta)

	app1 := &fleet.VPPApp{Name: "vpp_app_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "1", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b1"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	require.NoError(t, err)

	app2 := &fleet.VPPApp{Name: "vpp_app_2", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "2", Platform: fleet.IOSPlatform}}, BundleIdentifier: "b2"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app2, &team1.ID)
	require.NoError(t, err)

	app3 := &fleet.VPPApp{Name: "vpp_app_3", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "3", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b3"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app3, &team2.ID)
	require.NoError(t, err)

	vpp1, err := ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	require.NoError(t, err)

	vpp2, err := ds.InsertVPPAppWithTeam(ctx, app2, &team1.ID)
	require.NoError(t, err)

	vpp3, err := ds.InsertVPPAppWithTeam(ctx, app3, &team2.ID)
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE vpp_apps_teams SET install_during_setup = 1 WHERE adam_id IN (?, ?, ?)", vpp1.AdamID, vpp2.AdamID, vpp3.AdamID)
		return err
	})

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 2)
	assert.Equal(t, vpp1.AdamID, titles[1].AppStoreApp.AppStoreID)
	assert.Equal(t, 2, count)
	assert.NotNil(t, meta)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, team2.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 2)
	assert.Equal(t, vpp3.AdamID, titles[1].AppStoreApp.AppStoreID)
	assert.Equal(t, 2, count)
	assert.NotNil(t, meta)
}

func testSetSetupExperienceTitles(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	installerID1, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		UninstallScript:   "goodbye",
		InstallerFile:     bytes.NewReader([]byte("hello")),
		StorageID:         "storage1",
		Filename:          "file1",
		Title:             "file1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
	})
	_ = installerID1
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
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
	})
	_ = installerID2
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
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.MacOSPlatform),
	})
	_ = installerID3
	require.NoError(t, err)

	installerID4, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "pear",
		PreInstallQuery:   "SELECT 4",
		PostInstallScript: "apple",
		InstallerFile:     bytes.NewReader([]byte("hello2")),
		StorageID:         "storage3",
		Filename:          "file4",
		Title:             "file4",
		Version:           "4.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.IOSPlatform),
	})
	_ = installerID4
	require.NoError(t, err)

	titles, count, meta, err := ds.ListSetupExperienceSoftwareTitles(ctx, team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 0)
	assert.Equal(t, 0, count)
	assert.NotNil(t, meta)

	app1 := &fleet.VPPApp{Name: "vpp_app_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "1", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b1"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	require.NoError(t, err)

	app2 := &fleet.VPPApp{Name: "vpp_app_2", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "2", Platform: fleet.IOSPlatform}}, BundleIdentifier: "b2"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app2, &team1.ID)
	require.NoError(t, err)

	app3 := &fleet.VPPApp{Name: "vpp_app_3", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "3", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b3"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app3, &team2.ID)
	require.NoError(t, err)

	vpp1, err := ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	_ = vpp1
	require.NoError(t, err)

	vpp2, err := ds.InsertVPPAppWithTeam(ctx, app2, &team1.ID)
	_ = vpp2
	require.NoError(t, err)

	vpp3, err := ds.InsertVPPAppWithTeam(ctx, app3, &team2.ID)
	_ = vpp3
	require.NoError(t, err)

	titleSoftware := make(map[string]uint)
	titleVPP := make(map[string]uint)

	softwareTitles, _, _, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: &team1.ID}, fleet.TeamFilter{TeamID: &team1.ID})
	require.NoError(t, err)

	for _, title := range softwareTitles {
		if title.AppStoreApp != nil {
			titleVPP[title.AppStoreApp.AppStoreID] = title.ID
		} else if title.SoftwarePackage != nil {
			titleSoftware[title.SoftwarePackage.Name] = title.ID
		}
	}

	softwareTitles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: &team2.ID}, fleet.TeamFilter{TeamID: &team2.ID})
	require.NoError(t, err)

	for _, title := range softwareTitles {
		if title.AppStoreApp != nil {
			titleVPP[title.AppStoreApp.AppStoreID] = title.ID
		} else if title.SoftwarePackage != nil {
			titleSoftware[title.SoftwarePackage.Name] = title.ID
		}
	}

	err = ds.SetSetupExperienceSoftwareTitles(ctx, team1.ID, []uint{titleSoftware["file1"]})
	require.NoError(t, err)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 1)
	assert.Equal(t, 1, count)
	assert.Equal(t, "file1", titles[0].SoftwarePackage.Name)
	assert.NotNil(t, meta)

	err = ds.SetSetupExperienceSoftwareTitles(ctx, team1.ID, []uint{titleVPP["1"]})
	require.NoError(t, err)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, titles, 1)
	require.Equal(t, 1, count)
	assert.Equal(t, "1", titles[0].AppStoreApp.AppStoreID)
	assert.NotNil(t, meta)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, team2.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, titles, 0)
	require.Equal(t, 0, count)
	require.NotNil(t, meta)

	// Assign one vpp and one installer app
	err = ds.SetSetupExperienceSoftwareTitles(ctx, team1.ID, []uint{titleVPP["1"], titleSoftware["file1"]})
	require.NoError(t, err)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 2)
	assert.Equal(t, 2, count)
	assert.NotNil(t, meta)

	// iOS software
	err = ds.SetSetupExperienceSoftwareTitles(ctx, team2.ID, []uint{titleSoftware["file4"]})
	require.ErrorContains(t, err, "unsupported")

	// ios vpp app
	err = ds.SetSetupExperienceSoftwareTitles(ctx, team1.ID, []uint{titleVPP["2"]})
	require.ErrorContains(t, err, "unsupported")

	// wrong team
	err = ds.SetSetupExperienceSoftwareTitles(ctx, team1.ID, []uint{titleVPP["3"]})
	require.ErrorContains(t, err, "not available")

	// good other team assignment
	err = ds.SetSetupExperienceSoftwareTitles(ctx, team2.ID, []uint{titleVPP["3"]})
	require.NoError(t, err)

	// non-existent title ID
	err = ds.SetSetupExperienceSoftwareTitles(ctx, team1.ID, []uint{999})
	require.ErrorContains(t, err, "not available")

	// Failures and other team assignments didn't affected the number of apps on team 1
	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 2)
	assert.Equal(t, 2, count)
	assert.NotNil(t, meta)
}

func testSetupExperienceStatusResults(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	hostUUID := uuid.NewString()

	// Create a software installer
	// We need a new user first
	user, err := ds.NewUser(ctx, &fleet.User{Name: "Foo", Email: "foo@example.com", GlobalRole: ptr.String("admin"), Password: []byte("12characterslong!")})
	require.NoError(t, err)
	installerID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{Filename: "test.app", Version: "1.0.0", UserID: user.ID})
	require.NoError(t, err)
	installer, err := ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
	require.NoError(t, err)

	// VPP setup: create a token so that we can insert a VPP app
	dataToken, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), "Donkey Kong", "Jungle")
	require.NoError(t, err)
	tok1, err := ds.InsertVPPToken(ctx, dataToken)
	assert.NoError(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tok1.ID, []uint{})
	assert.NoError(t, err)
	vppApp, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{BundleIdentifier: "com.test.test", Name: "test.app", LatestVersion: "1.0.0"}, nil)
	require.NoError(t, err)
	var vppAppsTeamsID uint
	err = sqlx.GetContext(context.Background(), ds.reader(ctx),
		&vppAppsTeamsID, `SELECT id FROM vpp_apps_teams WHERE adam_id = ?`,
		vppApp.AdamID,
	)
	require.NoError(t, err)

	// TODO: use DS methods once those are written
	var scriptID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx, `INSERT INTO setup_experience_scripts (name) VALUES (?)`,
			"test_script")
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		scriptID = uint(id)
		return nil
	})

	insertSetupExperienceStatusResult := func(sesr *fleet.SetupExperienceStatusResult) {
		stmt := `INSERT INTO setup_experience_status_results (id, host_uuid, name, status, software_installer_id, host_software_installs_id, vpp_app_team_id, nano_command_uuid, setup_experience_script_id, script_execution_id, error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			res, err := q.ExecContext(ctx, stmt,
				sesr.ID, sesr.HostUUID, sesr.Name, sesr.Status, sesr.SoftwareInstallerID, sesr.HostSoftwareInstallsID, sesr.VPPAppTeamID, sesr.NanoCommandUUID, sesr.SetupExperienceScriptID, sesr.ScriptExecutionID, sesr.Error)
			require.NoError(t, err)
			id, err := res.LastInsertId()
			require.NoError(t, err)
			sesr.ID = uint(id)
			return nil
		})
	}

	expRes := []*fleet.SetupExperienceStatusResult{
		{
			HostUUID:            hostUUID,
			Name:                "software",
			Status:              fleet.SetupExperienceStatusPending,
			SoftwareInstallerID: ptr.Uint(installerID),
			SoftwareTitleID:     installer.TitleID,
		},
		{
			HostUUID:        hostUUID,
			Name:            "vpp",
			Status:          fleet.SetupExperienceStatusPending,
			VPPAppTeamID:    ptr.Uint(vppAppsTeamsID),
			SoftwareTitleID: ptr.Uint(vppApp.TitleID),
		},
		{
			HostUUID:                hostUUID,
			Name:                    "script",
			Status:                  fleet.SetupExperienceStatusPending,
			SetupExperienceScriptID: ptr.Uint(scriptID),
		},
	}

	for _, r := range expRes {
		insertSetupExperienceStatusResult(r)
	}

	res, err := ds.ListSetupExperienceResultsByHostUUID(ctx, hostUUID)
	require.NoError(t, err)
	require.Len(t, res, 3)
	for i, s := range expRes {
		require.Equal(t, s, res[i])
	}
}
