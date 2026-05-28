package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMDMIdPAccountsReconciliation(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"AssociateHostMDMIdPAccountTriggersReconciliation", testAssociateHostMDMIdPAccountTriggersReconciliation},
		{"AndroidEnrollmentFlowWithIdP", testAndroidEnrollmentFlowWithIdP},
		{"AndroidDeleteAndReEnrollPopulatesDeviceMapping", testAndroidDeleteAndReEnrollPopulatesDeviceMapping},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// testAssociateHostMDMIdPAccountTriggersReconciliation verifies that calling AssociateHostMDMIdPAccount
// triggers email reconciliation for ANY platform (our change that fixes Android IdP)
func testAssociateHostMDMIdPAccountTriggersReconciliation(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Android, macOS, Windows, and Linux to verify cross-platform
	// behavior. Windows and Linux pin down issue #45066: the Orbit Setup
	// Experience SSO callback (shared by Windows MSI / Linux Orbit enrollment
	// with End User Authentication).
	//
	// This test ensures the full AssociateHostMDMIdPAccount
	// populates host_emails on every supported platform.
	platforms := []struct {
		name     string
		platform string
		uuid     string
	}{
		{"Android", "android", "android-host-uuid"},
		{"macOS", "darwin", "macos-host-uuid"}, // Apple platforms
		{"Windows", "windows", "windows-host-uuid"},
		{"Linux", "ubuntu", "linux-host-uuid"},
	}

	// create IdP account
	idpAccount := &fleet.MDMIdPAccount{
		Username: "test.user",
		Fullname: "Test User",
		Email:    "test.user@example.com",
	}
	err := ds.InsertMDMIdPAccount(ctx, idpAccount)
	require.NoError(t, err)

	// get the generated UUID
	insertedAccount, err := ds.GetMDMIdPAccountByEmail(ctx, "test.user@example.com")
	require.NoError(t, err)
	require.NotNil(t, insertedAccount)
	idpAccount.UUID = insertedAccount.UUID

	for _, p := range platforms {
		t.Run(p.name, func(t *testing.T) {
			// Create host for this platform
			host := &fleet.Host{
				Hostname:      p.name + "-host",
				UUID:          p.uuid,
				Platform:      p.platform,
				OSVersion:     "Test OS",
				NodeKey:       ptr.String(p.uuid + "-key"),
				OsqueryHostID: ptr.String(p.uuid + "-osquery"),
			}

			h, err := ds.NewHost(ctx, host)
			require.NoError(t, err)
			require.NotZero(t, h.ID)

			// associate host with IdP account, should trigger reconciliation
			err = ds.AssociateHostMDMIdPAccount(ctx, p.uuid, idpAccount.UUID)
			require.NoError(t, err)

			// host_emails table has IdP email
			var emails []string
			err = ds.writer(ctx).SelectContext(ctx, &emails,
				`SELECT email FROM host_emails WHERE host_id = ? AND source = ?`,
				h.ID, fleet.DeviceMappingMDMIdpAccounts)
			require.NoError(t, err)
			require.Len(t, emails, 1, "Platform %s should have exactly one IdP email after association", p.name)
			assert.Equal(t, "test.user@example.com", emails[0])

			// calling again shouldn't create duplicates
			err = ds.AssociateHostMDMIdPAccount(ctx, p.uuid, idpAccount.UUID)
			require.NoError(t, err)

			emails = nil
			err = ds.writer(ctx).SelectContext(ctx, &emails,
				`SELECT email FROM host_emails WHERE host_id = ? AND source = ?`,
				h.ID, fleet.DeviceMappingMDMIdpAccounts)
			require.NoError(t, err)
			require.Len(t, emails, 1, "Platform %s should still have exactly one IdP email after re-association", p.name)
			assert.Equal(t, "test.user@example.com", emails[0])
		})
	}
}

// testAndroidEnrollmentFlowWithIdP tests the complete Android enrollment flow
// as it happens in production: NewAndroidHost followed by AssociateHostMDMIdPAccount
func testAndroidEnrollmentFlowWithIdP(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create IdP account (during SSO login)
	idpAccount := &fleet.MDMIdPAccount{
		Username: "android.user",
		Fullname: "Android User",
		Email:    "android.user@example.com",
	}
	err := ds.InsertMDMIdPAccount(ctx, idpAccount)
	require.NoError(t, err)

	insertedAccount, err := ds.GetMDMIdPAccountByEmail(ctx, "android.user@example.com")
	require.NoError(t, err)
	require.NotNil(t, insertedAccount)
	idpAccount.UUID = insertedAccount.UUID

	// simulate Android device enrollment
	const enterpriseSpecificID = "android-device-001"
	androidHost := &fleet.AndroidHost{
		Host: &fleet.Host{
			Hostname:       "Android Device",
			ComputerName:   "Pixel 8",
			Platform:       "android",
			OSVersion:      "Android 14",
			Build:          "UP1A.231005.007",
			Memory:         8192,
			HardwareSerial: "SERIAL123",
			CPUType:        "arm64",
			HardwareModel:  "Pixel 8",
			HardwareVendor: "Google",
			UUID:           "android-uuid-001",
		},
		Device: &android.Device{
			DeviceID:             "device-001",
			EnterpriseSpecificID: ptr.String(enterpriseSpecificID),
		},
	}
	androidHost.SetNodeKey(enterpriseSpecificID)

	// simulates enrollment call
	result, err := ds.NewAndroidHost(ctx, androidHost, false)
	require.NoError(t, err)
	require.NotZero(t, result.Host.ID)

	// no emails yet
	var emails []string
	err = ds.writer(ctx).SelectContext(ctx, &emails,
		`SELECT email FROM host_emails WHERE host_id = ? AND source = ?`,
		result.Host.ID, fleet.DeviceMappingMDMIdpAccounts)
	require.NoError(t, err)
	require.Empty(t, emails, "No IdP emails should exist immediately after NewAndroidHost")

	// associate with IdP account
	err = ds.AssociateHostMDMIdPAccount(ctx, "android-uuid-001", idpAccount.UUID)
	require.NoError(t, err)

	// verify reconciliation happened
	err = ds.writer(ctx).SelectContext(ctx, &emails,
		`SELECT email FROM host_emails WHERE host_id = ? AND source = ?`,
		result.Host.ID, fleet.DeviceMappingMDMIdpAccounts)
	require.NoError(t, err)
	require.Len(t, emails, 1)
	assert.Equal(t, "android.user@example.com", emails[0])

	// the host record (for username field in the future)
	host, err := ds.Host(ctx, result.Host.ID)
	require.NoError(t, err)
	require.NotNil(t, host)
	// N.b.: if/when username field is added to hosts table, verify it here
}

// testAndroidDeleteAndReEnrollPopulatesDeviceMapping reproduces the customer
// scenario from issue #43278: an Android host enrolled with IdP info, then
// deleted from Fleet (without unenrolling), then re-enrolled. The hosts list
// device_mapping was returning null for the re-enrolled host because
// host_mdm_idp_accounts (keyed by host_uuid) was preserved across deletion
// while host_emails (keyed by host_id) was cleared, leaving the two tables
// inconsistent on re-enrollment.
func testAndroidDeleteAndReEnrollPopulatesDeviceMapping(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	idpAccount := &fleet.MDMIdPAccount{
		Username: "pixel.user",
		Fullname: "Pixel User",
		Email:    "pixel.user@example.com",
	}
	require.NoError(t, ds.InsertMDMIdPAccount(ctx, idpAccount))
	insertedAccount, err := ds.GetMDMIdPAccountByEmail(ctx, "pixel.user@example.com")
	require.NoError(t, err)
	require.NotNil(t, insertedAccount)
	idpAccount.UUID = insertedAccount.UUID

	const hostUUID = "pixel9-uuid"
	newAndroidHost := func(nodeKey string) *fleet.AndroidHost {
		h := &fleet.AndroidHost{
			Host: &fleet.Host{
				Hostname:       "Pixel 9",
				ComputerName:   "Pixel 9",
				Platform:       "android",
				OSVersion:      "Android 14",
				Build:          "UP1A.231005.007",
				Memory:         8192,
				HardwareSerial: "PIXEL9SERIAL",
				CPUType:        "arm64",
				HardwareModel:  "Pixel 9",
				HardwareVendor: "Google",
				UUID:           hostUUID,
			},
			Device: &android.Device{
				DeviceID:             "device-pixel9",
				EnterpriseSpecificID: ptr.String(hostUUID),
			},
		}
		h.SetNodeKey(nodeKey)
		return h
	}

	// First enrollment: associate with IdP and verify host_emails is populated.
	first, err := ds.NewAndroidHost(ctx, newAndroidHost(hostUUID), false)
	require.NoError(t, err)
	require.NoError(t, ds.AssociateHostMDMIdPAccount(ctx, hostUUID, idpAccount.UUID))

	emails, err := ds.GetHostEmails(ctx, hostUUID, fleet.DeviceMappingMDMIdpAccounts)
	require.NoError(t, err)
	require.Equal(t, []string{"pixel.user@example.com"}, emails)

	// Delete the host (simulating the admin removing it from Fleet without
	// unenrolling first). host_mdm_idp_accounts must be cleared so that
	// re-enrollment treats the device as a fresh enrollment.
	require.NoError(t, ds.DeleteHost(ctx, first.Host.ID))

	var count int
	require.NoError(t, ds.writer(ctx).GetContext(ctx, &count,
		`SELECT COUNT(*) FROM host_mdm_idp_accounts WHERE host_uuid = ?`, hostUUID))
	require.Equal(t, 0, count, "host_mdm_idp_accounts must be cleared on Android host deletion")
	require.NoError(t, ds.writer(ctx).GetContext(ctx, &count,
		`SELECT COUNT(*) FROM host_emails WHERE host_id = ?`, first.Host.ID))
	require.Equal(t, 0, count, "host_emails must be cleared on host deletion")

	// Re-enrollment: same enterprise-specific ID -> same host UUID, but a new
	// hosts row with a new host_id. AssociateHostMDMIdPAccount must repopulate
	// host_emails so the hosts list endpoint returns device_mapping.
	second, err := ds.NewAndroidHost(ctx, newAndroidHost(hostUUID+"-2"), false)
	require.NoError(t, err)
	require.NotEqual(t, first.Host.ID, second.Host.ID, "re-enrollment should create a new hosts row")
	require.NoError(t, ds.AssociateHostMDMIdPAccount(ctx, hostUUID, idpAccount.UUID))

	emails, err = ds.GetHostEmails(ctx, hostUUID, fleet.DeviceMappingMDMIdpAccounts)
	require.NoError(t, err)
	require.Equal(t, []string{"pixel.user@example.com"}, emails,
		"host_emails must be repopulated on re-enrollment so the hosts list shows device_mapping")
}
