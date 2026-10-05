package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261005061457(t *testing.T) {
	db := applyUpToPrev(t)

	enrollmentID := execNoErrLastID(t, db, `INSERT INTO acme_enrollments (path_identifier, host_identifier) VALUES ('path', 'serial')`)
	accountID := execNoErrLastID(t, db, `INSERT INTO acme_accounts (acme_enrollment_id, json_web_key, json_web_key_thumbprint) VALUES (?, '{}', 'thumb')`, enrollmentID)
	orderID := execNoErrLastID(t, db, `INSERT INTO acme_orders (acme_account_id, certificate_signing_request, identifiers) VALUES (?, '', '[]')`, accountID)
	authzID := execNoErrLastID(t, db, `INSERT INTO acme_authorizations (identifier_type, identifier_value, acme_order_id) VALUES ('permanent-identifier', 'serial', ?)`, orderID)
	challengeID := execNoErrLastID(t, db, `INSERT INTO acme_challenges (challenge_type, token, acme_authorization_id, status) VALUES ('device-attest-01', 'token', ?, 'valid')`, authzID)

	applyNext(t, db)

	// challenges validated before the upgrade have no attested key on record
	var key []byte
	require.NoError(t, db.Get(&key, `SELECT attested_public_key FROM acme_challenges WHERE id = ?`, challengeID))
	require.Nil(t, key)

	execNoErr(t, db, `UPDATE acme_challenges SET attested_public_key = ? WHERE id = ?`, []byte{0x30, 0x59}, challengeID)
	require.NoError(t, db.Get(&key, `SELECT attested_public_key FROM acme_challenges WHERE id = ?`, challengeID))
	require.Equal(t, []byte{0x30, 0x59}, key)
}
