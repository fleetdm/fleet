package tables

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"

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

	// Assert the final schema using the production helpers. The test DB is selected per-connection (USE in newDBConnForTests), so this
	// read-only tx must be rolled back before any other db.* read to avoid forcing a second, database-less connection from the pool.
	tx, err := db.Begin()
	require.NoError(t, err)
	// The legacy text column must be gone and the new blob column present and NOT NULL.
	require.False(t, columnExists(tx, "windows_mdm_responses", "raw_response"), "raw_response must be dropped")
	require.True(t, columnExists(tx, "windows_mdm_responses", "raw_response_gz"), "raw_response_gz must exist")
	require.False(t, columnIsNullable(tx, "windows_mdm_responses", "raw_response_gz"), "raw_response_gz must be NOT NULL")
	require.NoError(t, tx.Rollback())

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
