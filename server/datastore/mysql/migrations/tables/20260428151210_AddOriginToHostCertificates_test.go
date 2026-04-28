package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260428151210(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a host so we can reference it from host_certificates.
	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?);`,
		"oh1", "nk1", "h1", "uuid-1", "darwin")
	var hostID uint
	require.NoError(t, db.Get(&hostID, `SELECT id FROM hosts WHERE uuid = 'uuid-1'`))

	// Insert an existing host_certificates row before migration. This stands in for
	// rows already present in production (origin should default to 'osquery').
	execNoErr(t, db, `
		INSERT INTO host_certificates (
			host_id, not_valid_after, not_valid_before, certificate_authority,
			common_name, key_algorithm, key_strength, key_usage,
			serial, signing_algorithm,
			subject_country, subject_org, subject_org_unit, subject_common_name,
			issuer_country, issuer_org, issuer_org_unit, issuer_common_name,
			sha1_sum
		) VALUES (?, '2027-01-01', '2026-01-01', 0, 'cn', 'rsa', 2048, 'digitalSignature',
			'1', 'sha256WithRSAEncryption', '', '', '', '', '', '', '', '',
			?)`,
		hostID, []byte("0123456789abcdef0123"))

	applyNext(t, db)

	// New origin column must exist with default 'osquery' for the pre-existing row.
	var origin string
	require.NoError(t, db.Get(&origin, `SELECT origin FROM host_certificates WHERE host_id = ?`, hostID))
	require.Equal(t, "osquery", origin)

	// Insert a new row explicitly tagged origin='mdm' to confirm the enum accepts both values.
	execNoErr(t, db, `
		INSERT INTO host_certificates (
			host_id, not_valid_after, not_valid_before, certificate_authority,
			common_name, key_algorithm, key_strength, key_usage,
			serial, signing_algorithm,
			subject_country, subject_org, subject_org_unit, subject_common_name,
			issuer_country, issuer_org, issuer_org_unit, issuer_common_name,
			sha1_sum, origin
		) VALUES (?, '2027-01-01', '2026-01-01', 0, 'cn', 'rsa', 2048, 'digitalSignature',
			'2', 'sha256WithRSAEncryption', '', '', '', '', '', '', '', '',
			?, 'mdm')`,
		hostID, []byte("fedcba9876543210fedc"))

	var origins []string
	require.NoError(t, db.Select(&origins, `SELECT origin FROM host_certificates WHERE host_id = ? ORDER BY serial`, hostID))
	require.Equal(t, []string{"osquery", "mdm"}, origins)
}
