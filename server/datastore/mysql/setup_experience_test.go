package mysql

import (
	"context"
	"database/sql"
	"strings"
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
		{"EnqueueSetupExperienceItems", testEnqueueSetupExperienceItems},
		{"GetSetupExperienceTitles", testGetSetupExperienceTitles},
		{"SetSetupExperienceTitles", testSetSetupExperienceTitles},
		{"ListSetupExperienceStatusResults", testSetupExperienceStatusResults},
		{"SetupExperienceScriptCRUD", testSetupExperienceScriptCRUD},
		{"TestHostInSetupExperience", testHostInSetupExperience},
		{"TestGetSetupExperienceScriptByID", testGetSetupExperienceScriptByID},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// TODO(JVE): this test could probably be simplified and most of the ad-hoc SQL removed.
func testEnqueueSetupExperienceItems(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	// Create some teams
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	team3, err := ds.NewTeam(ctx, &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// Create some software installers and add them to setup experience
	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		UninstallScript:   "goodbye",
		InstallerFile:     tfr1,
		StorageID:         "storage1",
		Filename:          "file1",
		Title:             "Software1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	tfr2, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "banana",
		PreInstallQuery:   "SELECT 3",
		PostInstallScript: "apple",
		InstallerFile:     tfr2,
		StorageID:         "storage3",
		Filename:          "file3",
		Title:             "Software2",
		Version:           "3.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id IN (?, ?)", installerID1, installerID2)
		return err
	})

	// Create some VPP apps and add them to setup experience
	app1 := &fleet.VPPApp{Name: "vpp_app_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "1", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b1"}
	vpp1, err := ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	require.NoError(t, err)

	app2 := &fleet.VPPApp{Name: "vpp_app_2", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "2", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b2"}
	vpp2, err := ds.InsertVPPAppWithTeam(ctx, app2, &team2.ID)
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE vpp_apps_teams SET install_during_setup = 1 WHERE adam_id IN (?, ?)", vpp1.AdamID, vpp2.AdamID)
		return err
	})

	// Create some scripts and add them to setup experience
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{Name: "script1", ScriptContents: "SCRIPT 1", TeamID: &team1.ID})
	require.NoError(t, err)
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{Name: "script2", ScriptContents: "SCRIPT 2", TeamID: &team2.ID})
	require.NoError(t, err)

	script1, err := ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)

	script2, err := ds.GetSetupExperienceScript(ctx, &team2.ID)
	require.NoError(t, err)

	h1 := newTestHostWithPlatform(t, ds, "123", "darwin", &team1.ID)
	h2 := newTestHostWithPlatform(t, ds, "456", "darwin", &team2.ID)
	h3 := newTestHostWithPlatform(t, ds, "789", "darwin", &team3.ID)

	hostTeam1 := h1.UUID
	hostTeam2 := h2.UUID
	hostTeam3 := h3.UUID

	anythingEnqueued, err := ds.EnqueueSetupExperienceItems(ctx, hostTeam1, team1.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)
	awaitingConfig, err := ds.GetHostAwaitingConfiguration(ctx, hostTeam1)
	require.NoError(t, err)
	require.True(t, awaitingConfig)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, hostTeam2, team2.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)
	awaitingConfig, err = ds.GetHostAwaitingConfiguration(ctx, hostTeam2)
	require.NoError(t, err)
	require.True(t, awaitingConfig)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, hostTeam3, team3.ID)
	require.NoError(t, err)
	require.False(t, anythingEnqueued)
	// Nothing is configured for setup experience in team 3, so we do not set
	// host_mdm_apple_awaiting_configuration.
	awaitingConfig, err = ds.GetHostAwaitingConfiguration(ctx, hostTeam3)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
	require.False(t, awaitingConfig)

	seRows := []setupExperienceInsertTestRows{}

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &seRows, "SELECT host_uuid, name, status, software_installer_id, setup_experience_script_id, vpp_app_team_id FROM setup_experience_status_results")
	})

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		DumpTable(t, q, "software_installer_labels")
		return nil
	})

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		var x struct {
			CIL uint `db:"count_installer_labels"`
			CHL uint `db:"count_host_labels"`
		}
		stmt := `		SELECT 0 AS count_installer_labels, 0 AS count_host_labels
		WHERE NOT EXISTS (
			SELECT 1 FROM software_installer_labels sil WHERE sil.software_installer_id = ?
		)
`
		if err := sqlx.GetContext(ctx, q, &x, stmt, installerID2); err != nil {
			return err
		}

		t.Logf("x: %v\n", x)
		return nil
	})

	require.Len(t, seRows, 6)

	for _, tc := range []setupExperienceInsertTestRows{
		{
			HostUUID:            hostTeam1,
			Name:                "Software1",
			Status:              "pending",
			SoftwareInstallerID: nullableUint(installerID1),
		},
		{
			HostUUID:            hostTeam2,
			Name:                "Software2",
			Status:              "pending",
			SoftwareInstallerID: nullableUint(installerID2),
		},
		{
			HostUUID:     hostTeam1,
			Name:         app1.Name,
			Status:       "pending",
			VPPAppTeamID: nullableUint(1),
		},
		{
			HostUUID:     hostTeam2,
			Name:         app2.Name,
			Status:       "pending",
			VPPAppTeamID: nullableUint(2),
		},
		{
			HostUUID: hostTeam1,
			Name:     "script1",
			Status:   "pending",
			ScriptID: nullableUint(script1.ID),
		},
		{
			HostUUID: hostTeam2,
			Name:     "script2",
			Status:   "pending",
			ScriptID: nullableUint(script2.ID),
		},
	} {
		var found bool
		for _, row := range seRows {
			if row == tc {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Couldn't find entry in setup_experience_status_results table: %#v", tc)
		}
	}

	require.Condition(t, func() (success bool) {
		for _, row := range seRows {
			if row.HostUUID == hostTeam3 {
				return false
			}
		}

		return true
	})

	// Remove team2's setup experience items
	err = ds.DeleteSetupExperienceScript(ctx, &team2.ID)
	require.NoError(t, err)

	err = ds.SetSetupExperienceSoftwareTitles(ctx, team2.ID, []uint{})
	require.NoError(t, err)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, hostTeam1, team1.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)

	// team2 now has nothing enqueued
	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, hostTeam2, team2.ID)
	require.NoError(t, err)
	require.False(t, anythingEnqueued)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, hostTeam3, team3.ID)
	require.NoError(t, err)
	require.False(t, anythingEnqueued)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &seRows, "SELECT host_uuid, name, status, software_installer_id, setup_experience_script_id, vpp_app_team_id FROM setup_experience_status_results")
	})

	require.Len(t, seRows, 3)

	for _, tc := range []setupExperienceInsertTestRows{
		{
			HostUUID:            hostTeam1,
			Name:                "Software1",
			Status:              "pending",
			SoftwareInstallerID: nullableUint(installerID1),
		},
		{
			HostUUID:     hostTeam1,
			Name:         app1.Name,
			Status:       "pending",
			VPPAppTeamID: nullableUint(1),
		},
		{
			HostUUID: hostTeam1,
			Name:     "script1",
			Status:   "pending",
			ScriptID: nullableUint(script1.ID),
		},
	} {
		var found bool
		for _, row := range seRows {
			if row == tc {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Couldn't find entry in setup_experience_status_results table: %#v", tc)
		}
	}

	for _, row := range seRows {
		if row.HostUUID == hostTeam3 || row.HostUUID == hostTeam2 {
			team := 2
			if row.HostUUID == hostTeam3 {
				team = 3
			}
			t.Errorf("team %d shouldn't have any any entries", team)
		}
	}
}

type setupExperienceInsertTestRows struct {
	HostUUID            string        `db:"host_uuid"`
	Name                string        `db:"name"`
	Status              string        `db:"status"`
	SoftwareInstallerID sql.NullInt64 `db:"software_installer_id"`
	ScriptID            sql.NullInt64 `db:"setup_experience_script_id"`
	VPPAppTeamID        sql.NullInt64 `db:"vpp_app_team_id"`
}

func nullableUint(val uint) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(val), Valid: true} // nolint: gosec
}

func testGetSetupExperienceTitles(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		UninstallScript:   "goodbye",
		InstallerFile:     tfr1,
		StorageID:         "storage1",
		Filename:          "file1",
		Title:             "file1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	tfr3, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID3, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "banana",
		PreInstallQuery:   "SELECT 3",
		PostInstallScript: "apple",
		InstallerFile:     tfr3,
		StorageID:         "storage3",
		Filename:          "file3",
		Title:             "file3",
		Version:           "3.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	tfr4, err := fleet.NewTempFileReader(strings.NewReader("hello2"), t.TempDir)
	require.NoError(t, err)
	installerID4, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "pear",
		PreInstallQuery:   "SELECT 4",
		PostInstallScript: "apple",
		InstallerFile:     tfr4,
		StorageID:         "storage3",
		Filename:          "file4",
		Title:             "file4",
		Version:           "4.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.IOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	titles, count, meta, err := ds.ListSetupExperienceSoftwareTitles(ctx, team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 1)
	assert.Equal(t, 1, count)
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

	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		UninstallScript:   "goodbye",
		InstallerFile:     tfr1,
		StorageID:         "storage1",
		Filename:          "file1",
		Title:             "file1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	_ = installerID1
	require.NoError(t, err)

	tfr2, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "world",
		PreInstallQuery:   "SELECT 2",
		PostInstallScript: "hello",
		InstallerFile:     tfr2,
		StorageID:         "storage2",
		Filename:          "file2",
		Title:             "file2",
		Version:           "2.0",
		Source:            "apps",
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	_ = installerID2
	require.NoError(t, err)

	tfr3, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID3, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "banana",
		PreInstallQuery:   "SELECT 3",
		PostInstallScript: "apple",
		InstallerFile:     tfr3,
		StorageID:         "storage3",
		Filename:          "file3",
		Title:             "file3",
		Version:           "3.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	_ = installerID3
	require.NoError(t, err)

	tfr4, err := fleet.NewTempFileReader(strings.NewReader("hello2"), t.TempDir)
	require.NoError(t, err)
	installerID4, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "pear",
		PreInstallQuery:   "SELECT 4",
		PostInstallScript: "apple",
		InstallerFile:     tfr4,
		StorageID:         "storage3",
		Filename:          "file4",
		Title:             "file4",
		Version:           "4.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.IOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	_ = installerID4
	require.NoError(t, err)

	titles, count, meta, err := ds.ListSetupExperienceSoftwareTitles(ctx, team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 2)
	assert.Equal(t, 2, count)
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

	// Single installer
	err = ds.SetSetupExperienceSoftwareTitles(ctx, team1.ID, []uint{titleSoftware["file1"]})
	require.NoError(t, err)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 3)
	assert.Equal(t, 3, count)
	assert.Equal(t, "file1", titles[0].SoftwarePackage.Name)
	assert.Equal(t, "file2", titles[1].SoftwarePackage.Name)
	assert.Equal(t, "1", titles[2].AppStoreApp.AppStoreID)
	assert.NotNil(t, meta)

	assert.True(t, *titles[0].SoftwarePackage.InstallDuringSetup)
	assert.False(t, *titles[1].SoftwarePackage.InstallDuringSetup)
	assert.False(t, *titles[2].AppStoreApp.InstallDuringSetup)

	// Single vpp app replaces installer
	err = ds.SetSetupExperienceSoftwareTitles(ctx, team1.ID, []uint{titleVPP["1"]})
	require.NoError(t, err)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, titles, 3)
	require.Equal(t, 3, count)
	assert.Equal(t, "file1", titles[0].SoftwarePackage.Name)
	assert.Equal(t, "file2", titles[1].SoftwarePackage.Name)
	assert.Equal(t, "1", titles[2].AppStoreApp.AppStoreID)
	assert.NotNil(t, meta)

	assert.False(t, *titles[0].SoftwarePackage.InstallDuringSetup)
	assert.False(t, *titles[1].SoftwarePackage.InstallDuringSetup)
	assert.True(t, *titles[2].AppStoreApp.InstallDuringSetup)

	// Team 2 unaffected
	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, team2.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, titles, 2)
	require.Equal(t, 2, count)
	assert.Equal(t, "file3", titles[0].SoftwarePackage.Name)
	assert.Equal(t, "3", titles[1].AppStoreApp.AppStoreID)
	require.NotNil(t, meta)

	assert.False(t, *titles[0].SoftwarePackage.InstallDuringSetup)
	assert.False(t, *titles[1].AppStoreApp.InstallDuringSetup)

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
	assert.Len(t, titles, 3)
	assert.Equal(t, 3, count)
	assert.NotNil(t, meta)

	// Empty slice removes all tiles
	err = ds.SetSetupExperienceSoftwareTitles(ctx, team1.ID, []uint{})
	require.NoError(t, err)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 3)
	assert.Equal(t, 3, count)
	assert.NotNil(t, meta)

	assert.False(t, *titles[0].SoftwarePackage.InstallDuringSetup)
	assert.False(t, *titles[1].SoftwarePackage.InstallDuringSetup)
	assert.False(t, *titles[2].AppStoreApp.InstallDuringSetup)
}

func testSetupExperienceStatusResults(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	hostUUID := uuid.NewString()

	// Create a software installer
	// We need a new user first
	user, err := ds.NewUser(ctx, &fleet.User{Name: "Foo", Email: "foo@example.com", GlobalRole: ptr.String("admin"), Password: []byte("12characterslong!")})
	require.NoError(t, err)
	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{Filename: "test.app", Version: "1.0.0", UserID: user.ID, ValidatedLabels: &fleet.LabelIdentsWithScope{}})
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
		scriptID = uint(id) // nolint: gosec
		return nil
	})

	insertSetupExperienceStatusResult := func(sesr *fleet.SetupExperienceStatusResult) {
		stmt := `INSERT INTO setup_experience_status_results (id, host_uuid, name, status, software_installer_id, host_software_installs_execution_id, vpp_app_team_id, nano_command_uuid, setup_experience_script_id, script_execution_id, error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			res, err := q.ExecContext(ctx, stmt,
				sesr.ID, sesr.HostUUID, sesr.Name, sesr.Status, sesr.SoftwareInstallerID, sesr.HostSoftwareInstallsExecutionID, sesr.VPPAppTeamID, sesr.NanoCommandUUID, sesr.SetupExperienceScriptID, sesr.ScriptExecutionID, sesr.Error)
			require.NoError(t, err)
			id, err := res.LastInsertId()
			require.NoError(t, err)
			sesr.ID = uint(id) // nolint: gosec
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

func testSetupExperienceScriptCRUD(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	// create a script for team1
	wantScript1 := &fleet.Script{
		Name:           "script",
		TeamID:         &team1.ID,
		ScriptContents: "echo foo",
	}

	err = ds.SetSetupExperienceScript(ctx, wantScript1)
	require.NoError(t, err)

	// get the script for team1
	gotScript1, err := ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)
	require.NotNil(t, gotScript1)
	require.Equal(t, wantScript1.Name, gotScript1.Name)
	require.Equal(t, wantScript1.TeamID, gotScript1.TeamID)
	require.NotZero(t, gotScript1.ScriptContentID)

	b, err := ds.GetAnyScriptContents(ctx, gotScript1.ScriptContentID)
	require.NoError(t, err)
	require.Equal(t, wantScript1.ScriptContents, string(b))

	// create a script for team2
	wantScript2 := &fleet.Script{
		Name:           "script",
		TeamID:         &team2.ID,
		ScriptContents: "echo bar",
	}

	err = ds.SetSetupExperienceScript(ctx, wantScript2)
	require.NoError(t, err)

	// get the script for team2
	gotScript2, err := ds.GetSetupExperienceScript(ctx, &team2.ID)
	require.NoError(t, err)
	require.NotNil(t, gotScript2)
	require.Equal(t, wantScript2.Name, gotScript2.Name)
	require.Equal(t, wantScript2.TeamID, gotScript2.TeamID)
	require.NotZero(t, gotScript2.ScriptContentID)
	require.NotEqual(t, gotScript1.ScriptContentID, gotScript2.ScriptContentID)

	b, err = ds.GetAnyScriptContents(ctx, gotScript2.ScriptContentID)
	require.NoError(t, err)
	require.Equal(t, wantScript2.ScriptContents, string(b))

	// create a script with no team id
	wantScriptNoTeam := &fleet.Script{
		Name:           "script",
		ScriptContents: "echo bar",
	}

	err = ds.SetSetupExperienceScript(ctx, wantScriptNoTeam)
	require.NoError(t, err)

	// get the script nil team id is equivalent to team id 0
	gotScriptNoTeam, err := ds.GetSetupExperienceScript(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, gotScriptNoTeam)
	require.Equal(t, wantScriptNoTeam.Name, gotScriptNoTeam.Name)
	require.Nil(t, gotScriptNoTeam.TeamID)
	require.NotZero(t, gotScriptNoTeam.ScriptContentID)
	require.Equal(t, gotScript2.ScriptContentID, gotScriptNoTeam.ScriptContentID) // should be the same as team2

	b, err = ds.GetAnyScriptContents(ctx, gotScriptNoTeam.ScriptContentID)
	require.NoError(t, err)
	require.Equal(t, wantScriptNoTeam.ScriptContents, string(b))

	// try to create another with name "script" and no team id
	var existsErr fleet.AlreadyExistsError
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{Name: "script", ScriptContents: "echo baz"})
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)

	// try to create another script with no team id and a different name
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{Name: "script2", ScriptContents: "echo baz"})
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)

	// try to add a script for a team that doesn't exist
	var fkErr fleet.ForeignKeyError
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{TeamID: ptr.Uint(42), Name: "script", ScriptContents: "echo baz"})
	require.Error(t, err)
	require.ErrorAs(t, err, &fkErr)

	// delete the script for team1
	err = ds.DeleteSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)

	// get the script for team1
	_, err = ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// try to delete script for team1 again
	err = ds.DeleteSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err) // TODO: confirm if we want to return not found on deletes

	// try to delete script for team that doesn't exist
	err = ds.DeleteSetupExperienceScript(ctx, ptr.Uint(42))
	require.NoError(t, err) // TODO: confirm if we want to return not found on deletes

	// add same script for team1 again
	err = ds.SetSetupExperienceScript(ctx, wantScript1)
	require.NoError(t, err)

	// get the script for team1
	oldScript1 := gotScript1
	newScript1, err := ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)
	require.NotNil(t, newScript1)
	require.Equal(t, wantScript1.Name, newScript1.Name)
	require.Equal(t, wantScript1.TeamID, newScript1.TeamID)
	require.NotZero(t, newScript1.ScriptContentID)
	// script contents are deleted by CleanupUnusedScriptContents not by DeleteSetupExperienceScript
	// so the content id should be the same as the old
	require.Equal(t, oldScript1.ScriptContentID, newScript1.ScriptContentID)
}

func testHostInSetupExperience(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	err := ds.SetHostAwaitingConfiguration(ctx, "abc", true)
	require.NoError(t, err)

	inSetupExperience, err := ds.GetHostAwaitingConfiguration(ctx, "abc")
	require.NoError(t, err)
	require.True(t, inSetupExperience)

	err = ds.SetHostAwaitingConfiguration(ctx, "abc", false)
	require.NoError(t, err)

	inSetupExperience, err = ds.GetHostAwaitingConfiguration(ctx, "abc")
	require.NoError(t, err)
	require.False(t, inSetupExperience)

	// host without a record in the table returns not found
	inSetupExperience, err = ds.GetHostAwaitingConfiguration(ctx, "404")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
	require.False(t, inSetupExperience)
}

func testGetSetupExperienceScriptByID(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	script := &fleet.Script{
		Name:           "setup_experience_script",
		ScriptContents: "echo hello",
	}

	err := ds.SetSetupExperienceScript(ctx, script)
	require.NoError(t, err)

	scriptByTeamID, err := ds.GetSetupExperienceScript(ctx, nil)
	require.NoError(t, err)

	gotScript, err := ds.GetSetupExperienceScriptByID(ctx, scriptByTeamID.ID)
	require.NoError(t, err)

	require.Equal(t, script.Name, gotScript.Name)
	require.NotZero(t, gotScript.ScriptContentID)

	b, err := ds.GetAnyScriptContents(ctx, gotScript.ScriptContentID)
	require.NoError(t, err)
	require.Equal(t, script.ScriptContents, string(b))
}
