package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20250609102714(t *testing.T) {
	db := applyUpToPrev(t)

	hostID := insertHost(t, db, nil)
	certID := execNoErrLastID(t, db, `INSERT INTO host_certificates (
		host_id,             not_valid_after,     not_valid_before, certificate_authority,
		common_name,         key_algorithm,       key_strength,     key_usage,
		serial,              signing_algorithm,   subject_country,  subject_org,
		subject_org_unit,    subject_common_name, issuer_country,   issuer_org,
		issuer_org_unit,     issuer_common_name,  sha1_sum
	) VALUES (
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?)`,
		hostID, time.Now(), time.Now(), false,
		"test-cert", "rsa", 2048, "digitalSignature",
		"1234567890", "sha256WithRSAEncryption", "US", "TestOrg",
		"TestUnit", "TestCommonName", "US", "TestOrg",
		"TestUnit", "TestIssuerCommonName", "test-sha1-sum")

	// Apply current migration.
	applyNext(t, db)

	var info struct {
		Source   string
		Username string
	}
	err := db.Get(&info, `SELECT source, username FROM host_certificate_sources WHERE host_certificate_id = ?`, certID)
	require.NoError(t, err)
	require.Equal(t, "system", info.Source)
	require.Equal(t, "", info.Username)
}
