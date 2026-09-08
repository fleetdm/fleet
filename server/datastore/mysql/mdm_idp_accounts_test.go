package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
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
		{"ReconcileSupersedesManuallySetIdPMapping", testReconcileSupersedesManuallySetIdPMapping},
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

// testReconcileSupersedesManuallySetIdPMapping verifies that reconciling a
// host's IdP association on (re-)enrollment replaces a manually set IdP
// mapping (source "idp", written by SetOrUpdateIDPHostDeviceMapping) instead
// of adding a second mapping reported under the same "mdm_idp_accounts" API
// source (issue #49914: duplicate device mapping after EACS wipe and ADE
// re-enrollment).
func testReconcileSupersedesManuallySetIdPMapping(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	idpAccount := &fleet.MDMIdPAccount{
		Username: "sso.user",
		Fullname: "SSO User",
		Email:    "sso.user@example.com",
	}
	err := ds.InsertMDMIdPAccount(ctx, idpAccount)
	require.NoError(t, err)

	insertedAccount, err := ds.GetMDMIdPAccountByEmail(ctx, "sso.user@example.com")
	require.NoError(t, err)
	require.NotNil(t, insertedAccount)
	idpAccount.UUID = insertedAccount.UUID

	newDarwinHost := func(t *testing.T, uuid string) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      uuid + "-hostname",
			UUID:          uuid,
			Platform:      "darwin",
			NodeKey:       new(uuid + "-key"),
			OsqueryHostID: new(uuid + "-osquery"),
		})
		require.NoError(t, err)
		require.NotZero(t, h.ID)
		return h
	}

	countRawIdpRows := func(t *testing.T, hostID uint) int {
		var count int
		err := sqlx.GetContext(ctx, ds.writer(ctx), &count,
			`SELECT COUNT(*) FROM host_emails WHERE host_id = ? AND source = ?`,
			hostID, fleet.DeviceMappingIDP)
		require.NoError(t, err)
		return count
	}

	t.Run("manual mapping with different email", func(t *testing.T) {
		host := newDarwinHost(t, "uuid-manual-diff")

		err := ds.SetOrUpdateIDPHostDeviceMapping(ctx, host.ID, "someone.else@example.com")
		require.NoError(t, err)

		// enrollment reconciles the association and supersedes the manual mapping
		err = ds.AssociateHostMDMIdPAccount(ctx, host.UUID, idpAccount.UUID)
		require.NoError(t, err)

		mappings, err := ds.ListHostDeviceMapping(ctx, host.ID)
		require.NoError(t, err)
		require.Len(t, mappings, 1)
		assert.Equal(t, "sso.user@example.com", mappings[0].Email)
		assert.Equal(t, fleet.DeviceMappingMDMIdpAccounts, mappings[0].Source)
		assert.Zero(t, countRawIdpRows(t, host.ID))
	})

	t.Run("manual mapping with same email", func(t *testing.T) {
		host := newDarwinHost(t, "uuid-manual-same")

		err := ds.SetOrUpdateIDPHostDeviceMapping(ctx, host.ID, "sso.user@example.com")
		require.NoError(t, err)

		err = ds.AssociateHostMDMIdPAccount(ctx, host.UUID, idpAccount.UUID)
		require.NoError(t, err)

		mappings, err := ds.ListHostDeviceMapping(ctx, host.ID)
		require.NoError(t, err)
		require.Len(t, mappings, 1)
		assert.Equal(t, "sso.user@example.com", mappings[0].Email)
		assert.Equal(t, fleet.DeviceMappingMDMIdpAccounts, mappings[0].Source)
		assert.Zero(t, countRawIdpRows(t, host.ID))
	})

	t.Run("correct mapping kept and stray manual mapping removed", func(t *testing.T) {
		host := newDarwinHost(t, "uuid-hit-plus-manual")

		// a correct enrollment row already exists alongside a stray manual row
		// with a different email
		_, err := ds.writer(ctx).ExecContext(ctx,
			`INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?), (?, ?, ?)`,
			host.ID, "sso.user@example.com", fleet.DeviceMappingMDMIdpAccounts,
			host.ID, "someone.else@example.com", fleet.DeviceMappingIDP)
		require.NoError(t, err)

		err = ds.AssociateHostMDMIdPAccount(ctx, host.UUID, idpAccount.UUID)
		require.NoError(t, err)

		mappings, err := ds.ListHostDeviceMapping(ctx, host.ID)
		require.NoError(t, err)
		require.Len(t, mappings, 1)
		assert.Equal(t, "sso.user@example.com", mappings[0].Email)
		assert.Equal(t, fleet.DeviceMappingMDMIdpAccounts, mappings[0].Source)
		assert.Zero(t, countRawIdpRows(t, host.ID))
	})

	t.Run("existing duplicate mapping is healed on re-enrollment", func(t *testing.T) {
		host := newDarwinHost(t, "uuid-dup")

		// manufacture the duplicate state from #49914: a manual "idp" row alongside
		// an enrollment "mdm_idp_accounts" row with the same email
		_, err := ds.writer(ctx).ExecContext(ctx,
			`INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?), (?, ?, ?)`,
			host.ID, "sso.user@example.com", fleet.DeviceMappingIDP,
			host.ID, "sso.user@example.com", fleet.DeviceMappingMDMIdpAccounts)
		require.NoError(t, err)

		err = ds.AssociateHostMDMIdPAccount(ctx, host.UUID, idpAccount.UUID)
		require.NoError(t, err)

		mappings, err := ds.ListHostDeviceMapping(ctx, host.ID)
		require.NoError(t, err)
		require.Len(t, mappings, 1)
		assert.Equal(t, "sso.user@example.com", mappings[0].Email)
		assert.Equal(t, fleet.DeviceMappingMDMIdpAccounts, mappings[0].Source)
		assert.Zero(t, countRawIdpRows(t, host.ID))
	})

	t.Run("manual mapping survives reconcile without idp account", func(t *testing.T) {
		host := newDarwinHost(t, "uuid-manual-no-idp")

		err := ds.SetOrUpdateIDPHostDeviceMapping(ctx, host.ID, "someone.else@example.com")
		require.NoError(t, err)

		// no host_mdm_idp_accounts association exists (e.g. re-enrollment without SSO)
		err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			_, err := reconcileHostEmailsFromMdmIdpAccountsDB(ctx, tx, ds.logger, host.ID)
			return err
		})
		require.NoError(t, err)

		mappings, err := ds.ListHostDeviceMapping(ctx, host.ID)
		require.NoError(t, err)
		require.Len(t, mappings, 1)
		assert.Equal(t, "someone.else@example.com", mappings[0].Email)
		// a lone manual mapping is still reported under the translated
		// "mdm_idp_accounts" source
		assert.Equal(t, fleet.DeviceMappingMDMIdpAccounts, mappings[0].Source)
		assert.Equal(t, 1, countRawIdpRows(t, host.ID))
	})
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
