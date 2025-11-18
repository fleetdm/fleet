package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
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

	// create VPP apps
	app1, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "android1", BundleIdentifier: "android1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "something_android_app_1", Platform: fleet.AndroidPlatform},
			Configuration: json.RawMessage(`{"ManagedConfiguration": {"DisableShareScreen": true, "DisableComputerAudio": true}}`),
		}}, &team1.ID)
	require.NoError(t, err)

	app2, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.app.vpp1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_spaceeee_1", Platform: fleet.IOSPlatform},
			Configuration: json.RawMessage(`{"ManagedConfiguration": {"ios app shouldn't have configuration": true}}`),
		}}, &team1.ID)
	require.NoError(t, err)

	// get android app
	meta, err := ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, app1.TitleID)
	require.NoError(t, err)
	require.NotZero(t, meta.VPPAppsTeamsID)
	require.Equal(t, "android1", meta.BundleIdentifier)

	// get ios app
	meta2, err := ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, app2.TitleID)
	require.NoError(t, err)
	require.NotZero(t, meta2.VPPAppsTeamsID)
	// require.Equal(t, "{blablabla}", meta.Configuration) TODO(JK): this should return configuration

	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		DumpTable(t, tx, "vpp_apps")
		DumpTable(t, tx, "android_app_configurations")
		return nil
	})
}

func testAndroidAppConfigValidation(t *testing.T, ds *Datastore) {

	cases := []struct {
		desc    string
		config  json.RawMessage
		wantErr string
	}{
		{
			desc:    "empty",
			config:  json.RawMessage(""),
			wantErr: "EOF",
		},
		{
			desc:   "empty tree",
			config: json.RawMessage("{}"),
			// wantErr: "this probably should give an error",
		},
		{
			desc:    "invalid json",
			config:  json.RawMessage(`{"ManagedConfiguration": {"DisableShareScreen": true, "DisableComputerAudio": true}xyz}`),
			wantErr: "invalid character 'x' after object key:value pair",
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
			config: json.RawMessage(`"workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED"`),
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
		fmt.Println(c.desc)
		err := validateAndroidAppConfiguration(c.config)
		if c.wantErr != "" {
			require.EqualError(t, err, c.wantErr)
		}
	}

}
