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

	// Android and one Apple platform to verify cross-platform behavior
	platforms := []struct {
		name     string
		platform string
		uuid     string
	}{
		{"Android", "android", "android-host-uuid"},
		{"macOS", "darwin", "macos-host-uuid"}, // Apple platforms
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
	result, err := ds.NewAndroidHost(ctx, androidHost)
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
