package fleet

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestAppConfigGet(t *testing.T) {
	c := &AppConfigPayload{
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
	require.Equal(t, json.RawMessage{}, c.GetJSON("host_settings.additional_queries"))
}
