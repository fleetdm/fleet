package fleet

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestAppConfigGet(t *testing.T) {
	c := &AppConfig{
		OrgInfo: &OrgInfo{
			OrgName: ptr.String("somename"),
		},
		HostSettings: &HostSettings{
			EnableSoftwareInventory: ptr.Bool(true),
		},
	}
	require.Equal(t, "somename", c.GetString("org_info.org_name"))
	require.Equal(t, true, c.GetBool("host_settings.enable_software_inventory"))

	// check defaults
	require.Equal(t, true, c.GetBool("host_settings.enable_host_users"))

	// check undefined zero reasonably
	require.Equal(t, "", c.GetString("org_info.org_logo_url"))
	require.Equal(t, json.RawMessage(nil), c.GetJSON("host_settings.additional_queries"))

	// returns zero/default when types mismatch
	require.Equal(t, json.RawMessage(nil), c.GetJSON("org_info.org_logo_url"))

	// return zero for type when path doesn't exist
	require.Equal(t, 0, c.GetInt("some.non.existent.path"))
	require.Equal(t, json.RawMessage(nil), c.GetJSON("org_info.asdfasdf"))

	// gets default if parent struct is not defined
	require.Equal(t, false, c.GetBool("host_expiry_settings.host_expiry_enabled"))
}
