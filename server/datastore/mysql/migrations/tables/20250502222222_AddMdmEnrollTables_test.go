package tables

import (
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20250502222222(t *testing.T) {
	db := applyUpToPrev(t)

	type hostEmail struct {
		HostID uint   `db:"host_id"`
		Email  string `db:"email"`
		Source string `db:"source"`
	}

	type hostMDM struct {
		HostID           uint   `db:"host_id"`
		FleetEnrollRef   string `db:"fleet_enroll_ref"`
		Enrolled         bool   `db:"enrolled"`
		ServerURL        string `db:"server_url"`
		InstalledFromDEP bool   `db:"installed_from_dep"`
		MDMID            uint   `db:"mdm_id"`
		IsServer         bool   `db:"is_server"`
	}

	type mdmIDPAccount struct {
		UUID     string `db:"uuid"`
		Email    string `db:"email"`
		Username string `db:"username"`
		Fullname string `db:"fullname"`
	}

	// type legacyEnrollRef struct {
	// 	HostUUID  string `db:"host_uuid"`
	// 	EnrollRef string `db:"enroll_ref"`
	// }

	type legacyAccount struct {
		ID             uint      `db:"id"`
		HostUUID       string    `db:"host_uuid"`
		HostID         uint      `db:"host_id"`
		Email          string    `db:"email"`
		EmailID        uint      `db:"email_id"`
		EmailCreatedAt time.Time `db:"email_created_at"`
		EmailUpdatedAt time.Time `db:"email_updated_at"`
		AccountUUID    string    `db:"account_uuid"`
	}

	// type hostMDMAccount struct {
	// 	HostUUID    string `db:"host_uuid"`
	// 	AccountUUID string `db:"account_uuid"`
	// }

	newHost := func(id uint, platform string) uint {
		return uint(execNoErrLastID(t, db, //nolint:gosec // dismiss G115
			`INSERT INTO hosts (hardware_serial, osquery_host_id, node_key, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
			fmt.Sprintf("serial-%d", id),
			fmt.Sprintf("osquery-host-id-%d", id),
			fmt.Sprintf("node-key-%d", id),
			fmt.Sprintf("host-uuid-%d", id),
			platform,
		))
	}

	newAccount := func(acct mdmIDPAccount) {
		execNoErr(t, db,
			`INSERT INTO mdm_idp_accounts (uuid, email, username, fullname) VALUES (?, ?, ?, ?)`,
			acct.UUID, acct.Email, acct.Username, acct.Fullname,
		)
	}
	newEmail := func(he hostEmail) {
		execNoErr(t, db,
			`INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
			he.HostID, he.Email, he.Source,
		)
	}

	newHostMDM := func(hmdm hostMDM) {
		execNoErr(t, db,
			`INSERT INTO host_mdm (host_id, fleet_enroll_ref, enrolled, server_url, installed_from_dep, mdm_id, is_server) 
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
			hmdm.HostID, hmdm.FleetEnrollRef, hmdm.Enrolled, hmdm.ServerURL,
			hmdm.InstalledFromDEP, hmdm.MDMID, hmdm.IsServer,
		)
	}

	bob := mdmIDPAccount{Username: "bob", Email: "bob@example.com", UUID: "bob-uuid", Fullname: "Bob"}
	alice := mdmIDPAccount{Username: "alice", Email: "alice@exmaple.com", UUID: "alice-uuid", Fullname: "Alice"}
	carol := mdmIDPAccount{Username: "carol", Email: "carol@example.com", UUID: "carol-uuid", Fullname: "Carol"}
	dave := mdmIDPAccount{Username: "dave", Email: "dave@example.com", UUID: "dave-uuid", Fullname: "Dave"}

	byEmail := make(map[string]mdmIDPAccount, 4)
	for _, acct := range []mdmIDPAccount{alice, carol, dave, bob} {
		newAccount(acct)
		byEmail[acct.Email] = acct
	}

	type testCase struct {
		name                 string
		hostID               uint
		hostUUID             string
		hostMDM              hostMDM
		hostEmails           []hostEmail
		expectLegacyRefs     []string
		expectLegacyEmails   []string
		expectMDMAccountUUID string
	}

	testCases := []testCase{
		{
			name:     "host with unmatched url ref but mdm account email match",
			hostID:   1,
			hostUUID: "host-uuid-1",
			hostMDM: hostMDM{
				HostID:         1,
				FleetEnrollRef: "nobody-uuid",
			},
			hostEmails: []hostEmail{
				{HostID: 1, Email: bob.Email, Source: fleet.DeviceMappingMDMIdpAccounts},
				{HostID: 1, Email: "nobody@example.com", Source: fleet.DeviceMappingMDMIdpAccounts},
			},
			expectLegacyRefs:     []string{"nobody-uuid"},
			expectLegacyEmails:   []string{bob.Email, "nobody@example.com"},
			expectMDMAccountUUID: bob.UUID, // legacy ref didn't match but we matched bob as an alternative
		},
		{
			name:     "host with unmatched url ref but multiple mdm account email matches",
			hostID:   2,
			hostUUID: "host-uuid-2",
			hostMDM: hostMDM{
				HostID:         2,
				FleetEnrollRef: "nobody-uuid",
			},
			hostEmails: []hostEmail{
				{HostID: 2, Email: dave.Email, Source: fleet.DeviceMappingMDMIdpAccounts},
				{HostID: 2, Email: bob.Email, Source: fleet.DeviceMappingMDMIdpAccounts},
				{HostID: 2, Email: "nobody@example.com", Source: fleet.DeviceMappingMDMIdpAccounts},
			},
			expectLegacyRefs:   []string{"nobody-uuid"},
			expectLegacyEmails: []string{bob.Email, dave.Email, "nobody@example.com"},
			// we arbitrarily pick the alphanumerically largest email when multiple matches created
			// at the same time
			expectMDMAccountUUID: dave.UUID,
		},
		{
			name:     "host with legacy match and multiple mdm account email matches",
			hostID:   3,
			hostUUID: "host-uuid-3",
			hostMDM: hostMDM{
				HostID:         3,
				FleetEnrollRef: "bob-uuid",
			},
			hostEmails: []hostEmail{
				{HostID: 3, Email: carol.Email, Source: fleet.DeviceMappingMDMIdpAccounts},
				{HostID: 3, Email: bob.Email, Source: fleet.DeviceMappingMDMIdpAccounts},
				{HostID: 3, Email: "nobody@example.com", Source: fleet.DeviceMappingMDMIdpAccounts},
			},
			expectLegacyRefs:   []string{"bob-uuid"},
			expectLegacyEmails: []string{carol.Email, bob.Email, "nobody@example.com"},
			// we'll set the host_emails entry for bob to occur before the entry for carol
			// to test that we prefer the most recent email over legacy refs
			expectMDMAccountUUID: carol.UUID,
		},
		{
			name:     "host with legacy match and no mdm account email matches",
			hostID:   4,
			hostUUID: "host-uuid-4",
			hostMDM: hostMDM{
				HostID:         4,
				FleetEnrollRef: "bob-uuid",
			},
			hostEmails: []hostEmail{
				{HostID: 4, Email: "nobody@example.com", Source: fleet.DeviceMappingMDMIdpAccounts},
			},
			expectLegacyRefs:     []string{"bob-uuid"},
			expectLegacyEmails:   []string{"nobody@example.com"},
			expectMDMAccountUUID: bob.UUID,
		},
		{
			name:     "host with no legacy match and no mdm account email matches",
			hostID:   5,
			hostUUID: "host-uuid-5",
			hostMDM: hostMDM{
				HostID:         5,
				FleetEnrollRef: "nobody-uuid",
			},
			hostEmails: []hostEmail{
				{HostID: 5, Email: "nobody@example.com", Source: fleet.DeviceMappingMDMIdpAccounts},
			},
			expectLegacyRefs:     []string{"nobody-uuid"},
			expectLegacyEmails:   []string{"nobody@example.com"},
			expectMDMAccountUUID: "", // no mdm account
		},
		{
			name:     "host with no legacy match and no mdm account emails",
			hostID:   6,
			hostUUID: "host-uuid-6",
			hostMDM: hostMDM{
				HostID:         6,
				FleetEnrollRef: "nobody-uuid",
			},
			hostEmails:           []hostEmail{},
			expectLegacyRefs:     []string{"nobody-uuid"},
			expectLegacyEmails:   []string{},
			expectMDMAccountUUID: "", // no mdm account
		},
		{
			name:     "host with legacy match and no emails",
			hostID:   7,
			hostUUID: "host-uuid-7",
			hostMDM: hostMDM{
				HostID:         7,
				FleetEnrollRef: "bob-uuid",
			},
			hostEmails:           []hostEmail{},
			expectLegacyRefs:     []string{"bob-uuid"},
			expectLegacyEmails:   []string{},
			expectMDMAccountUUID: bob.UUID,
		},
		{
			name:     "host with no legacy match and only google emails",
			hostID:   8,
			hostUUID: "host-uuid-8",
			hostMDM: hostMDM{
				HostID:         8,
				FleetEnrollRef: "nobody-uuid",
			},
			hostEmails: []hostEmail{
				{HostID: 8, Email: "bob@example.com", Source: fleet.DeviceMappingGoogleChromeProfiles},
			},
			expectLegacyRefs:     []string{"nobody-uuid"},
			expectLegacyEmails:   []string{}, // only included if source is fleet.DeviceMappingMDMIdpAccounts
			expectMDMAccountUUID: "",         // no mdm account
		},
	}

	for _, tc := range testCases {
		newHost(tc.hostID, "darwin")
		newHostMDM(hostMDM{
			HostID:           tc.hostID,
			FleetEnrollRef:   tc.hostMDM.FleetEnrollRef,
			Enrolled:         true,
			ServerURL:        "https://example.com",
			InstalledFromDEP: true,
			MDMID:            1,
			IsServer:         false,
		})
		for _, he := range tc.hostEmails {
			newEmail(he)
		}
	}

	// for host 3, set the host_emails entry for bob to occur before the entry for carol
	// this is to test that we prefer the most recent email even when there is a matching legacy ref
	execNoErr(t, db,
		`UPDATE host_emails SET created_at = ? WHERE host_id = ? AND email = ?`,
		time.Now().Add(-24*time.Hour), 3, bob.Email,
	)

	// Apply current migration.
	applyNext(t, db)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Check legacy enroll refs
			var legacyEnrollRefs []string
			require.NoError(t, db.Select(&legacyEnrollRefs,
				`SELECT enroll_ref FROM legacy_host_mdm_enroll_refs WHERE host_uuid = ? ORDER BY enroll_ref ASC`, tc.hostUUID))
			require.Len(t, legacyEnrollRefs, len(tc.expectLegacyRefs))
			require.Equal(t, tc.expectLegacyRefs, legacyEnrollRefs)

			// Check legacy accounts
			var legacyAccounts []legacyAccount
			require.NoError(t, db.Select(&legacyAccounts, `
SELECT 
	id, host_uuid, host_id, 
	email, email_id, email_created_at, email_updated_at, 
	coalesce(account_uuid, '') as account_uuid 
FROM 
	legacy_host_mdm_idp_accounts 
WHERE
	host_uuid = ?
ORDER BY email ASC`, tc.hostUUID))
			require.Len(t, legacyAccounts, len(tc.expectLegacyEmails))

			expectedByEmail := make(map[string]legacyAccount, len(tc.expectLegacyEmails))
			for _, email := range tc.expectLegacyEmails {
				ea := legacyAccount{
					HostUUID: tc.hostUUID,
					HostID:   tc.hostID,
					Email:    email,
				}
				if acct, ok := byEmail[email]; ok {
					ea.AccountUUID = acct.UUID
				}
				expectedByEmail[email] = ea
			}

			for _, la := range legacyAccounts {
				expected, ok := expectedByEmail[la.Email]
				require.True(t, ok, "unexpected legacy account %s", la.Email)
				require.Equal(t, expected.HostUUID, la.HostUUID)
				require.Equal(t, expected.HostID, la.HostID)
				require.Equal(t, expected.Email, la.Email)
				require.Equal(t, expected.AccountUUID, la.AccountUUID)
				delete(expectedByEmail, la.Email)
			}
			require.Empty(t, expectedByEmail, "missing legacy accounts %v", expectedByEmail)

			// Check host_mdm_idp_accounts
			var accountUUIDs []string
			require.NoError(t, db.Select(&accountUUIDs,
				`SELECT account_uuid FROM host_mdm_idp_accounts WHERE host_uuid = ?`, tc.hostUUID))
			if tc.expectMDMAccountUUID == "" {
				require.Empty(t, accountUUIDs)
			} else {
				require.Len(t, accountUUIDs, 1)
				require.Equal(t, tc.expectMDMAccountUUID, accountUUIDs[0])
			}
		})
	}
}
