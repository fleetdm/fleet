package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240302111134(t *testing.T) {
	testBatchSize = 2
	defer func() { testBatchSize = 0 }()

	db := applyUpToPrev(t)

	scriptA, scriptB, scriptC, scriptD := "scriptA", "scriptB", "scriptC", "scriptD"
	md5A, md5B, md5C, md5D := md5ChecksumScriptContent(scriptA),
		md5ChecksumScriptContent(scriptB),
		md5ChecksumScriptContent(scriptC),
		md5ChecksumScriptContent(scriptD)

	// create saved scripts for A and B
	savedScriptAID := execNoErrLastID(t, db, `INSERT INTO scripts (name, script_contents) VALUES (?, ?)`, "A", scriptA)
	savedScriptBID := execNoErrLastID(t, db, `INSERT INTO scripts (name, script_contents) VALUES (?, ?)`, "B", scriptB)

	// create some host executions for A and C (C being anonymous), none for B
	execNoErr(t, db, `INSERT INTO host_script_results (host_id, execution_id, output, script_id, script_contents) VALUES (1, uuid(), 'ok', ?, ?)`, savedScriptAID, scriptA)
	execNoErr(t, db, `INSERT INTO host_script_results (host_id, execution_id, output, script_id, script_contents) VALUES (1, uuid(), 'ok', ?, ?)`, savedScriptAID, scriptA)
	execNoErr(t, db, `INSERT INTO host_script_results (host_id, execution_id, output, script_id, script_contents) VALUES (2, uuid(), 'ok', ?, ?)`, savedScriptAID, scriptA)
	execNoErr(t, db, `INSERT INTO host_script_results (host_id, execution_id, output, script_contents) VALUES (3, uuid(), 'ok', ?)`, scriptC)
	// also create one for scriptD associated with savedScriptAID (possible if
	// that saved script was edited after execution)
	execNoErr(t, db, `INSERT INTO host_script_results (host_id, execution_id, output, script_id, script_contents) VALUES (4, uuid(), 'ok', ?, ?)`, savedScriptAID, scriptD)

	applyNext(t, db)

	// there should be 4 scripts in script_contents
	type scriptContent struct {
		ID          uint   `db:"id"`
		MD5Checksum string `db:"md5_checksum"`
		Contents    string `db:"contents"`
	}
	var scriptContents []*scriptContent
	err := db.Select(&scriptContents, `SELECT id, HEX(md5_checksum) as md5_checksum, contents FROM script_contents`)
	require.NoError(t, err)

	// build a lookup map of contents hash to script_content_id
	contentHashToID := make(map[string]uint, len(scriptContents))
	for _, sc := range scriptContents {
		contentHashToID[sc.MD5Checksum] = sc.ID
		sc.ID = 0
	}

	// check that the received script contents have the expected hash
	expect := []*scriptContent{
		{0, md5A, scriptA},
		{0, md5B, scriptB},
		{0, md5C, scriptC},
		{0, md5D, scriptD},
	}
	require.ElementsMatch(t, expect, scriptContents)

	// verify that the script_content_id of the other tables has been properly set
	var scriptContentID uint
	err = db.Get(&scriptContentID, `SELECT script_content_id FROM scripts WHERE id = ?`, savedScriptAID)
	require.NoError(t, err)
	require.Equal(t, contentHashToID[md5A], scriptContentID)

	err = db.Get(&scriptContentID, `SELECT script_content_id FROM scripts WHERE id = ?`, savedScriptBID)
	require.NoError(t, err)
	require.Equal(t, contentHashToID[md5B], scriptContentID)

	// hosts 1 and 2 have scriptA, host 3 has scriptC, host 4 has scriptD
	var hostResultIDs []struct {
		ScriptContentID *uint `db:"script_content_id"`
	}
	err = db.Select(&hostResultIDs, `SELECT script_content_id FROM host_script_results WHERE host_id IN (1, 2)`)
	require.NoError(t, err)
	// 3 rows, all the id of scriptA
	require.Len(t, hostResultIDs, 3)
	require.NotNil(t, hostResultIDs[0].ScriptContentID)
	require.NotNil(t, hostResultIDs[1].ScriptContentID)
	require.NotNil(t, hostResultIDs[2].ScriptContentID)
	require.Equal(t, contentHashToID[md5A], *hostResultIDs[0].ScriptContentID)
	require.Equal(t, contentHashToID[md5A], *hostResultIDs[1].ScriptContentID)
	require.Equal(t, contentHashToID[md5A], *hostResultIDs[2].ScriptContentID)

	hostResultIDs = nil
	err = db.Select(&hostResultIDs, `SELECT script_content_id FROM host_script_results WHERE host_id = 3`)
	require.NoError(t, err)
	require.Len(t, hostResultIDs, 1)
	require.NotNil(t, hostResultIDs[0].ScriptContentID)
	require.Equal(t, contentHashToID[md5C], *hostResultIDs[0].ScriptContentID)

	hostResultIDs = nil
	err = db.Select(&hostResultIDs, `SELECT script_content_id FROM host_script_results WHERE host_id = 4`)
	require.NoError(t, err)
	require.Len(t, hostResultIDs, 1)
	require.NotNil(t, hostResultIDs[0].ScriptContentID)
	require.Equal(t, contentHashToID[md5D], *hostResultIDs[0].ScriptContentID)
}
