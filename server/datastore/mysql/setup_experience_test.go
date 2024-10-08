package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestSetupExperience(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"EnqueueSetupExperienceItems", testEnqueueSetupExperienceItems},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testEnqueueSetupExperienceItems(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	team3, err := ds.NewTeam(ctx, &fleet.Team{Name: "team3"})
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
		Title:             "Software1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
	})
	require.NoError(t, err)

	installerID2, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "banana",
		PreInstallQuery:   "SELECT 3",
		PostInstallScript: "apple",
		InstallerFile:     bytes.NewReader([]byte("hello")),
		StorageID:         "storage3",
		Filename:          "file3",
		Title:             "Software2",
		Version:           "3.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.MacOSPlatform),
	})
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id IN (?, ?)", installerID1, installerID2)
		return err
	})

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

	var script1ID, script2ID int64
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		res, err := insertScriptContents(ctx, q, "SCRIPT 1")
		if err != nil {
			return err
		}
		id1, _ := res.LastInsertId()
		res, err = insertScriptContents(ctx, q, "SCRIPT 2")
		if err != nil {
			return err
		}
		id2, _ := res.LastInsertId()

		res, err = q.ExecContext(ctx, "INSERT INTO setup_experience_scripts (team_id, global_or_team_id, name, script_content_id) VALUES (?, ?, ?, ?)", team1.ID, team1.ID, "script1", id1)
		if err != nil {
			return err
		}
		script1ID, _ = res.LastInsertId()

		res, err = q.ExecContext(ctx, "INSERT INTO setup_experience_scripts (team_id, global_or_team_id, name, script_content_id) VALUES (?, ?, ?, ?)", team2.ID, team2.ID, "script2", id2)
		if err != nil {
			return err
		}
		script2ID, _ = res.LastInsertId()

		return nil
	})

	hostTeam1 := "123"
	hostTeam2 := "456"
	hostTeam3 := "789"

	anything, err := ds.EnqueueSetupExperienceItems(ctx, hostTeam1, team1.ID)
	require.NoError(t, err)
	require.True(t, anything)

	anything, err = ds.EnqueueSetupExperienceItems(ctx, hostTeam2, team2.ID)
	require.NoError(t, err)
	require.True(t, anything)

	anything, err = ds.EnqueueSetupExperienceItems(ctx, hostTeam3, team3.ID)
	require.NoError(t, err)
	require.False(t, anything)

	seRows := []setupExperienceInsertTestRows{}

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &seRows, "SELECT host_uuid, name, software_installer_id, setup_experience_script_id, vpp_app_team_id FROM setup_experience_status_results")
	})

	for _, row := range seRows {
		fmt.Printf("row: %#v\n", row)
	}

	for _, tc := range []setupExperienceInsertTestRows{
		{
			HostUUID:            hostTeam1,
			Name:                "Software1",
			SoftwareInstallerID: nullableUint(installerID1),
		},
		{
			HostUUID:            hostTeam2,
			Name:                "Software2",
			SoftwareInstallerID: nullableUint(installerID2),
		},
		{
			HostUUID:     hostTeam1,
			Name:         app1.Name,
			VPPAppTeamID: nullableUint(1),
		},
		{
			HostUUID:     hostTeam2,
			Name:         app2.Name,
			VPPAppTeamID: nullableUint(2),
		},
		{
			HostUUID: hostTeam1,
			Name:     "script1",
			ScriptID: nullableUint(uint(script1ID)),
		},
		{
			HostUUID: hostTeam2,
			Name:     "script2",
			ScriptID: nullableUint(uint(script2ID)),
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
}

type setupExperienceInsertTestRows struct {
	HostUUID            string        `db:"host_uuid"`
	Name                string        `db:"name"`
	SoftwareInstallerID sql.NullInt64 `db:"software_installer_id"`
	ScriptID            sql.NullInt64 `db:"setup_experience_script_id"`
	VPPAppTeamID        sql.NullInt64 `db:"vpp_app_team_id"`
}

func nullableUint(val uint) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(val), Valid: true}
}
