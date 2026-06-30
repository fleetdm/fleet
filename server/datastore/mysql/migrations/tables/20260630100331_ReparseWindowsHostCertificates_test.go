package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260630100331(t *testing.T) {
	db := applyUpToPrev(t)

	insertHost := func(platform, uuid string) uint {
		execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?);`,
			uuid, uuid, uuid, uuid, platform)
		var id uint
		require.NoError(t, db.Get(&id, `SELECT id FROM hosts WHERE uuid = ?`, uuid))
		return id
	}

	// insertCert inserts a host_certificates row (deletedAt nil => live) and
	// returns its id. sha1 values must be 20 bytes (binary(20)).
	insertCert := func(hostID uint, serial, origin string, sha1 []byte, deletedAt any) uint {
		execNoErr(t, db, `
			INSERT INTO host_certificates (
				host_id, not_valid_after, not_valid_before, certificate_authority,
				common_name, key_algorithm, key_strength, key_usage,
				serial, signing_algorithm,
				subject_country, subject_org, subject_org_unit, subject_common_name,
				issuer_country, issuer_org, issuer_org_unit, issuer_common_name,
				sha1_sum, origin, deleted_at
			) VALUES (?, '2027-01-01', '2026-01-01', 0, 'cn', 'rsa', 2048, 'digitalSignature',
				?, 'sha256WithRSAEncryption', '', '', '', '', '', '', '', '',
				?, ?, ?)`,
			hostID, serial, sha1, origin, deletedAt)
		var id uint
		require.NoError(t, db.Get(&id, `SELECT id FROM host_certificates WHERE sha1_sum = ?`, sha1))
		return id
	}

	winHost := insertHost("windows", "win-uuid")
	macHost := insertHost("darwin", "mac-uuid")

	// Windows osquery-origin live certs: must be soft-deleted by the migration so
	// they re-parse on the next ingestion.
	winOsq1 := insertCert(winHost, "1", "osquery", []byte("aaaaaaaaaaaaaaaaaaaa"), nil)
	winOsq2 := insertCert(winHost, "2", "osquery", []byte("bbbbbbbbbbbbbbbbbbbb"), nil)
	// Windows mdm-origin cert: parsed directly from the cert, must be left untouched.
	winMDM := insertCert(winHost, "3", "mdm", []byte("cccccccccccccccccccc"), nil)
	// Windows osquery cert already soft-deleted: must stay deleted (not re-touched).
	winDeleted := insertCert(winHost, "4", "osquery", []byte("dddddddddddddddddddd"), "2026-01-01 00:00:00.000000")
	// macOS osquery cert: unaffected by the Windows DN gap, must be left untouched.
	macOsq := insertCert(macHost, "5", "osquery", []byte("eeeeeeeeeeeeeeeeeeee"), nil)

	applyNext(t, db)

	isDeleted := func(id uint) bool {
		var deleted bool
		require.NoError(t, db.Get(&deleted, `SELECT deleted_at IS NOT NULL FROM host_certificates WHERE id = ?`, id))
		return deleted
	}

	// Windows osquery-origin live certs are now soft-deleted.
	require.True(t, isDeleted(winOsq1))
	require.True(t, isDeleted(winOsq2))
	// Windows MDM-origin and macOS certs are untouched.
	require.False(t, isDeleted(winMDM))
	require.False(t, isDeleted(macOsq))
	// Already-deleted Windows cert is still deleted.
	require.True(t, isDeleted(winDeleted))
}
