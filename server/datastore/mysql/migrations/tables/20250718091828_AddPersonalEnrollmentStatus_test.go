package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20250718091828(t *testing.T) {
	db := applyUpToPrev(t)

	//
	// Insert data to test the migration
	//
	// ...
	// host_id 1 is a
	_, err := db.DB.Exec(`INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_server, fleet_enroll_ref)
	VALUES (1, 1, 'https://example.com', 0, 0, ''), -- manually enrolled
	(2, 1, 'https://example.com', 1, 0, ''),  -- automatically enrolled from DEP, no enroll ref
	(3, 1, 'https://example.com', 1, 0, 'fleet-enroll-ref3'), -- automatically enrolled from DEP with enroll ref
	(4, 0, 'https://example.com', 0, 1, ''), -- server with MDM off
	(5, 0, 'https://example.com', 1, 0, ''), -- not yet enrolled device, but from DEP
	(6, 0, 'https://example.com', 0, 0, '') -- not enrolled device, not from DEP
	`)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	type hostMDM struct {
		HostID               uint    `db:"host_id"`
		EnrollmentStatus     *string `db:"enrollment_status"`
		Enrolled             bool    `db:"enrolled"`
		ServerURL            string  `db:"server_url"`
		InstalledFromDEP     bool    `db:"installed_from_dep"`
		IsServer             bool    `db:"is_server"`
		FleetEnrollRef       string  `db:"fleet_enroll_ref"`
		IsPersonalEnrollment bool    `db:"is_personal_enrollment"`
	}

	hostMDMEntries := []*hostMDM{}
	err = sqlx.SelectContext(t.Context(), db, &hostMDMEntries, `SELECT host_id, enrollment_status, enrolled, server_url, installed_from_dep, is_server, fleet_enroll_ref, is_personal_enrollment FROM host_mdm`)
	require.NoError(t, err)
	require.Len(t, hostMDMEntries, 6)
	for _, entry := range hostMDMEntries {
		assert.False(t, entry.IsPersonalEnrollment, "is_personal_enrollment should be false on all rows from before migration")
	}

	_, err = db.DB.Exec(`INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_server, fleet_enroll_ref, is_personal_enrollment)
	VALUES (7, 1, 'https://example.com', 0, 0, '', 1), 
	(8, 0, 'https://example.com', 0, 0, '', 1) -- personal enrollment turned off
    `)
	require.NoError(t, err)

	err = sqlx.SelectContext(t.Context(), db, &hostMDMEntries, `SELECT host_id, enrollment_status, enrolled, server_url, installed_from_dep, is_server, fleet_enroll_ref, is_personal_enrollment FROM host_mdm`)
	require.NoError(t, err)
	require.Len(t, hostMDMEntries, 8)
	for _, entry := range hostMDMEntries {
		if entry.HostID <= 6 {
			assert.False(t, entry.IsPersonalEnrollment, "is_personal_enrollment should be false on all rows from before migration")
		}
		switch entry.HostID {
		case 1:
			assert.Equal(t, "On (manual)", *entry.EnrollmentStatus)
		case 2:
			assert.Equal(t, "On (automatic)", *entry.EnrollmentStatus)
		case 3:
			assert.Equal(t, "On (automatic)", *entry.EnrollmentStatus)
		case 4:
			assert.Equal(t, (*string)(nil), entry.EnrollmentStatus)
		case 5:
			assert.Equal(t, "Pending", *entry.EnrollmentStatus)
		case 6:
			assert.Equal(t, "Off", *entry.EnrollmentStatus)
		case 7:
			assert.Truef(t, entry.IsPersonalEnrollment, "is_personal_enrollment should be true for host_id %d", entry.HostID)
			assert.Equal(t, "On (personal)", *entry.EnrollmentStatus)
		case 8:
			assert.Truef(t, entry.IsPersonalEnrollment, "is_personal_enrollment should be true for host_id %d", entry.HostID)
			assert.Equal(t, "Off", *entry.EnrollmentStatus)
		default:
			t.Fatalf("unexpected host_id %d", entry.HostID)
		}
	}
}
