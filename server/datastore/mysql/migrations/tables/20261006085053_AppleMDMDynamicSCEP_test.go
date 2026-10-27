package tables

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20261006085053(t *testing.T) {
	db := applyUpToPrev(t)

	notValidAfter := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO acme_enrollments (path_identifier, host_identifier, not_valid_after, revoked) VALUES
		('path-1', 'host-1', ?, 0),
		('path-2', 'host-2', NULL, 1)`, notValidAfter)
	require.NoError(t, err)

	applyNext(t, db)

	type enrollment struct {
		PathIdentifier string         `db:"path_identifier"`
		HostIdentifier string         `db:"host_identifier"`
		NotValidAfter  sql.NullTime   `db:"not_valid_after"`
		Revoked        bool           `db:"revoked"`
		Purpose        string         `db:"purpose"`
		EnrollmentID   sql.NullString `db:"enrollment_id"`
	}
	var got []enrollment
	err = db.Select(&got, `SELECT path_identifier, host_identifier, not_valid_after, revoked, purpose, enrollment_id
		FROM acme_enrollments ORDER BY path_identifier`)
	require.NoError(t, err)
	require.Equal(t, []enrollment{
		{"path-1", "host-1", sql.NullTime{Time: notValidAfter, Valid: true}, false, "acme", sql.NullString{}},
		{"path-2", "host-2", sql.NullTime{}, true, "acme", sql.NullString{}},
	}, got)

	// New renewal rows can carry the enrollment ID.
	_, err = db.Exec(`INSERT INTO acme_enrollments (path_identifier, host_identifier, purpose, enrollment_id)
		VALUES ('path-3', 'host-3', 'acme_renewal', 'enroll-3')`)
	require.NoError(t, err)
	var renewal enrollment
	err = db.Get(&renewal, `SELECT path_identifier, host_identifier, not_valid_after, revoked, purpose, enrollment_id
		FROM acme_enrollments WHERE path_identifier = 'path-3'`)
	require.NoError(t, err)
	require.Equal(t, "acme_renewal", renewal.Purpose)
	require.Equal(t, sql.NullString{String: "enroll-3", Valid: true}, renewal.EnrollmentID)

	insertChallenge := `INSERT INTO mdm_apple_scep_challenges (challenge, purpose, host_uuid, expires_at, issued_cert_serial)
		VALUES (?, 'ade', 'UDID1', DATE_ADD(NOW(6), INTERVAL 1 HOUR), ?)`
	execNoErr(t, db, insertChallenge, "abc", 1)
	// binary collation: a challenge differing only by case is a distinct value
	execNoErr(t, db, insertChallenge, "ABC", 2)
	// NULL serials don't collide with each other
	execNoErr(t, db, insertChallenge, "def", nil)
	execNoErr(t, db, insertChallenge, "ghi", nil)

	_, err = db.Exec(insertChallenge, "abc", 3)
	require.ErrorContains(t, err, "Duplicate entry")

	_, err = db.Exec(insertChallenge, "jkl", 1)
	require.ErrorContains(t, err, "Duplicate entry")

	var challenges []string
	require.NoError(t, db.Select(&challenges, `SELECT challenge FROM mdm_apple_scep_challenges ORDER BY id`))
	require.Equal(t, []string{"abc", "ABC", "def", "ghi"}, challenges)
}
