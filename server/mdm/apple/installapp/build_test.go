package installapp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildInstallApplicationXML_VPPManaged(t *testing.T) {
	xml, err := BuildInstallApplicationXML(Input{
		CommandUUID:     "cmd-1",
		ITunesStoreID:   "989804926",
		ManagementFlags: 1,
	})
	require.NoError(t, err)

	// Standard device enrollments must include ChangeManagementState=Managed.
	require.Contains(t, xml, "<key>ChangeManagementState</key>")
	require.Contains(t, xml, "<string>Managed</string>")

	require.Contains(t, xml, "<key>iTunesStoreID</key>")
	require.Contains(t, xml, "<integer>989804926</integer>")
	require.NotContains(t, xml, "ManifestURL")

	require.Contains(t, xml, "<key>ManagementFlags</key>")
	require.Contains(t, xml, "<integer>1</integer>")
	require.Contains(t, xml, "<key>RequestType</key>")
	require.Contains(t, xml, "<string>InstallApplication</string>")
	require.Contains(t, xml, "<key>InstallAsManaged</key>")
	require.Contains(t, xml, "<key>PurchaseMethod</key>")
	require.Contains(t, xml, "<key>CommandUUID</key>")
	require.Contains(t, xml, "<string>cmd-1</string>")
}

func TestBuildInstallApplicationXML_VPPUserEnrollment(t *testing.T) {
	xml, err := BuildInstallApplicationXML(Input{
		CommandUUID:      "cmd-bai",
		ITunesStoreID:    "989804926",
		ManagementFlags:  1,
		IsUserEnrollment: true,
	})
	require.NoError(t, err)

	// BYOD User Enrollments must NOT include ChangeManagementState — Apple
	// rejects InstallApplication commands that carry it.
	require.NotContains(t, xml, "ChangeManagementState")

	require.Contains(t, xml, "<key>iTunesStoreID</key>")
	require.Contains(t, xml, "<integer>989804926</integer>")
	require.Contains(t, xml, "<key>ManagementFlags</key>")
	require.Contains(t, xml, "<integer>1</integer>")
	require.Contains(t, xml, "<key>InstallAsManaged</key>")
	require.Contains(t, xml, "<string>cmd-bai</string>")
}

func TestBuildInstallApplicationXML_InHouseUserEnrollment(t *testing.T) {
	xml, err := BuildInstallApplicationXML(Input{
		CommandUUID:      "cmd-ipa",
		ManifestURL:      "https://fleet.example.com/api/latest/fleet/software/titles/42/in_house_app/manifest?fleet_id=0",
		ManagementFlags:  1,
		IsUserEnrollment: true,
	})
	require.NoError(t, err)

	require.NotContains(t, xml, "ChangeManagementState")
	require.NotContains(t, xml, "iTunesStoreID")

	require.Contains(t, xml, "<key>ManifestURL</key>")
	require.Contains(t, xml, "<string>https://fleet.example.com/api/latest/fleet/software/titles/42/in_house_app/manifest?fleet_id=0</string>")
	require.Contains(t, xml, "<string>cmd-ipa</string>")
}

func TestBuildInstallApplicationXML_InHouseManaged(t *testing.T) {
	xml, err := BuildInstallApplicationXML(Input{
		CommandUUID:     "cmd-ipa-managed",
		ManifestURL:     "https://fleet.example.com/manifest",
		ManagementFlags: 0,
	})
	require.NoError(t, err)

	require.Contains(t, xml, "<key>ChangeManagementState</key>")
	require.Contains(t, xml, "<string>Managed</string>")
	require.NotContains(t, xml, "iTunesStoreID")

	require.Contains(t, xml, "<key>ManifestURL</key>")
	require.Contains(t, xml, "<key>ManagementFlags</key>")
	require.Contains(t, xml, "<integer>0</integer>")
}

func TestBuildInstallApplicationXML_Validation(t *testing.T) {
	t.Run("empty CommandUUID rejected", func(t *testing.T) {
		_, err := BuildInstallApplicationXML(Input{ITunesStoreID: "1"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "CommandUUID")
	})
	t.Run("missing payload rejected", func(t *testing.T) {
		_, err := BuildInstallApplicationXML(Input{CommandUUID: "x"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "ITunesStoreID or ManifestURL")
	})
	t.Run("both payload fields rejected", func(t *testing.T) {
		_, err := BuildInstallApplicationXML(Input{
			CommandUUID:   "x",
			ITunesStoreID: "1",
			ManifestURL:   "https://example.com",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "mutually exclusive")
	})
}

func TestBuildInstallApplicationXML_EscapesAmpersandsInManifestURL(t *testing.T) {
	xml, err := BuildInstallApplicationXML(Input{
		CommandUUID:     "cmd",
		ManifestURL:     "https://fleet.example.com/manifest?fleet_id=0&team_id=42",
		ManagementFlags: 1,
	})
	require.NoError(t, err)
	require.Contains(t, xml, "fleet_id=0&amp;team_id=42")
	require.NotContains(t, xml, "fleet_id=0&team_id=42")
}

func TestBuildInstallApplicationXML_PlistShape(t *testing.T) {
	xml, err := BuildInstallApplicationXML(Input{
		CommandUUID:     "cmd-shape",
		ITunesStoreID:   "1",
		ManagementFlags: 0,
	})
	require.NoError(t, err)

	require.True(t, strings.HasPrefix(xml, "<?xml"))
	require.Contains(t, xml, `<!DOCTYPE plist`)
	require.Contains(t, xml, `<plist version="1.0">`)
	require.True(t, strings.HasSuffix(strings.TrimSpace(xml), "</plist>"))
}
