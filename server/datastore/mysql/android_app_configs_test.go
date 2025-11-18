package mysql

import (
	// "context"
	"encoding/json"
	"fmt"
	"testing"

	// "github.com/fleetdm/fleet/v4/server/fleet"
	// "github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestAndroidAppConfigs(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestAndroidAppConfigValidation", testAndroidAppConfigValidation},
		// {"TestAddDeleteAndroidAppConfig", testAndroidAppConfigCrud},
		// {"TestAddAppWithConfig", testAddAppWithAndroidConfig},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testAndroidAppConfigValidation(t *testing.T, ds *Datastore) {
	// ctx := context.Background()

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
			desc:    "invalid json",
			config:  json.RawMessage(`{"ManagedConfiguration": {"DisableShareScreen": true, "DisableComputerAudio": true}xyz}`),
			wantErr: "invalid character 'x' after object key:value pair",
		},
		{
			desc:   "valid json, managed configuration",
			config: json.RawMessage(`{"ManagedConfiguration": {"DisableShareScreen": true, "DisableComputerAudio": true}}`),
		},
		{
			desc:   "valid json, managed configuration",
			config: json.RawMessage(`"workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED"`),
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
