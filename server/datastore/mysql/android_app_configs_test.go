package mysql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestAndroidAppConfigs(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestAndroidAppConfigCrud", testAndroidAppConfigCrud},
		// {"TestAddAppWithConfig", testAddAppWithAndroidConfig},
		{"TestAndroidAppConfigValidation", testAndroidAppConfigValidation},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testAndroidAppConfigCrud(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	test.CreateInsertGlobalVPPToken(t, ds)

	// test cases: ios app, ios app with config, android app with no config, android app with config

	testConfig := json.RawMessage(`{"ManagedConfiguration": {"DisableShareScreen": true, "DisableComputerAudio": true}}`)
	// Create android and VPP apps
	app1, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "android1", BundleIdentifier: "android1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "something_android_app_1", Platform: fleet.AndroidPlatform},
			Configuration: testConfig,
		}}, &team1.ID)
	require.NoError(t, err)

	app2, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.app.vpp1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_forapple_1", Platform: fleet.IOSPlatform},
			Configuration: json.RawMessage(`{"ManagedConfiguration": {"ios app shouldn't have configuration": true}}`),
		}}, &team1.ID)
	require.NoError(t, err)

	// Get android app without team
	meta, err := ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, app1.TitleID)
	require.NoError(t, err)
	require.Zero(t, meta.Configuration)

	// Get android app and configuration
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team1.ID, app1.TitleID)
	require.NoError(t, err)
	require.NotZero(t, meta.VPPAppsTeamsID)
	require.NotZero(t, meta.Configuration)
	require.Equal(t, "android1", meta.BundleIdentifier)
	require.Equal(t, testConfig, meta.Configuration)

	// Get ios app
	meta2, err := ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, app2.TitleID)
	require.NoError(t, err)
	require.NotZero(t, meta2.VPPAppsTeamsID)

	// Edit android app
	newConfig := json.RawMessage(`{"workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED"}`)
	app1.VPPAppTeam.Configuration = newConfig
	_, err = ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	require.NoError(t, err)

	// Check that configuration was changed
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team1.ID, app1.TitleID)
	require.NoError(t, err)
	require.NotZero(t, meta.VPPAppsTeamsID)
	require.Equal(t, newConfig, meta.Configuration)

	// Add invalid configuration
	badConfig := json.RawMessage(`"-": "-"`)
	app1.VPPAppTeam.Configuration = badConfig
	_, err = ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	require.Error(t, err)

	// Delete app, should delete configuration
	require.NoError(t, ds.DeleteVPPAppFromTeam(ctx, &team1.ID, app1.VPPAppID))
	_, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team1.ID, app1.TitleID)
	require.ErrorContains(t, err, "not found")
	_, err = ds.GetAndroidAppConfiguration(ctx, app1.AdamID, team1.ID)
	require.ErrorContains(t, err, "not found")
}

func testAndroidAppConfigValidation(t *testing.T, ds *Datastore) {

	cases := []struct {
		desc    string
		config  json.RawMessage
		wantErr string
	}{
		{
			desc:    "null",
			config:  json.RawMessage(""),
			wantErr: "EOF",
		},
		{
			desc:   "empty tree",
			config: json.RawMessage("{}"),
			// wantErr: "this probably should give an error", or just be treated as a delete
		},
		{
			desc:    "invalid json",
			config:  json.RawMessage(`{"ManagedConfiguration": {"DisableShareScreen": true, "DisableComputerAudio": true}xyz}`),
			wantErr: "invalid character 'x' after object key:value pair",
		},
		{
			desc:    "invalid json, with key work profile widgets",
			config:  json.RawMessage(`"workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED"`),
			wantErr: "json: cannot unmarshal string into Go value of type mysql.androidAppConfig",
		},
		{
			desc:    "valid json, unknown key",
			config:  json.RawMessage(`"unknown": "key"`),
			wantErr: "json: cannot unmarshal string into Go value of type mysql.androidAppConfig",
		},
		{
			desc:   "valid json, managed configuration",
			config: json.RawMessage(`{"ManagedConfiguration": {"DisableShareScreen": true, "DisableComputerAudio": true}}`),
		},
		{
			desc:   "valid json, work profile widgets",
			config: json.RawMessage(`{"workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED"}`),
		},
		{
			desc:   "valid json, both",
			config: json.RawMessage(`{"managedConfiguration": {"test": "test"}, "workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED"}`),
		},
	}

	// configValidBoth?

	// TODO(JK): this needs to be mostly tested with uploading/editing/getting VPP app
	// as that is the only API to change configurations

	for _, c := range cases {
		t.Log("Running test case ", c.desc)
		err := validateAndroidAppConfiguration(c.config)
		if c.wantErr != "" {
			require.EqualError(t, err, c.wantErr)
		}
	}

}
