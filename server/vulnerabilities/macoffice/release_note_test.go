package macoffice_test

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/macoffice"
	"github.com/stretchr/testify/require"
)

func TestGetProductTypeFromBundleId(t *testing.T) {
	testCases := []struct {
		bundle string
		pType  macoffice.ProductType
		notOk  bool
	}{
		{
			bundle: "com.parallels.winapp.a5c41f715c1b8a880253846c025624e9.c23ed995b43c4ce1bd8d7ead2fa634fa",
			notOk:  true,
		},
		{
			bundle: "com.microsoft.teams",
			notOk:  true,
		},
		{
			bundle: "com.microsoft.Powerpoint",
			pType:  macoffice.PowerPoint,
		},
		{
			bundle: "com.microsoft.Word",
			pType:  macoffice.Word,
		},
		{
			bundle: "com.microsoft.Excel",
			pType:  macoffice.Excel,
		},
		{
			bundle: "com.microsoft.onenote.mac",
			pType:  macoffice.OneNote,
		},
		{
			// TODO: Need to make sure this is the right bundle
			bundle: "com.microsoft.outlook",
			pType:  macoffice.Outlook,
		},
	}

	for _, tc := range testCases {
		r, ok := macoffice.GetProductTypeFromBundleId(tc.bundle)
		if tc.notOk {
			require.False(t, ok)
		}
		require.Equal(t, tc.pType, r)
	}
}
