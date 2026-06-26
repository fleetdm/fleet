package tables

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20260626120000(t *testing.T) {
	db := applyUpToPrev(t)

	insertEnrollment := func(deviceID string) int64 {
		res, err := db.Exec(`INSERT INTO mdm_windows_enrollments
			(mdm_device_id, mdm_hardware_id, device_state, device_type, device_name, enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version)
			VALUES (?, ?, '', '', '', '', '', '', '')`, deviceID, deviceID+"-hw")
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	insertResponse := func(enrollID int64, raw string) int64 {
		res, err := db.Exec(`INSERT INTO windows_mdm_responses (enrollment_id, raw_response) VALUES (?, ?)`, enrollID, raw)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}

	enroll := insertEnrollment("deviceA")

	// A realistic, highly compressible envelope, a small one, and an empty value: the backfill must round-trip all three.
	largeEnvelope := `<SyncML xmlns="SYNCML:SYNCML1.2"><SyncHdr><VerDTD>1.2</VerDTD></SyncHdr><SyncBody>` +
		strings.Repeat(`<Status><CmdID>1</CmdID><Cmd>Replace</Cmd><Data>200</Data></Status>`, 200) + `</SyncBody></SyncML>`
	responses := map[int64]string{
		insertResponse(enroll, largeEnvelope): largeEnvelope,
		insertResponse(enroll, "<SyncML/>"):   "<SyncML/>",
		insertResponse(enroll, ""):            "",
	}

	applyNext(t, db)

	// The legacy text column must be gone and the new blob column present.
	require.False(t, columnExistsDB(t, db, "windows_mdm_responses", "raw_response"), "raw_response must be dropped")
	require.True(t, columnExistsDB(t, db, "windows_mdm_responses", "raw_response_gz"), "raw_response_gz must exist")

	// Every backfilled row must gunzip back to its original plaintext.
	for id, want := range responses {
		var stored []byte
		require.NoError(t, db.Get(&stored, `SELECT raw_response_gz FROM windows_mdm_responses WHERE id = ?`, id))

		gr, err := gzip.NewReader(bytes.NewReader(stored))
		require.NoError(t, err, "stored value for id %d must be valid gzip", id)
		got, err := io.ReadAll(gr)
		require.NoError(t, err)
		require.NoError(t, gr.Close())
		require.Equal(t, want, string(got), "round-trip mismatch for id %d", id)
	}
}

// columnExistsDB checks for a column using the sqlx test handle (the package-level columnExists takes a *sql.Tx).
func columnExistsDB(t *testing.T, db *sqlx.DB, table, column string) bool {
	var count int
	require.NoError(t, db.Get(&count, `
SELECT COUNT(*) FROM information_schema.columns
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`, table, column))
	return count > 0
}
