package tables

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20231212094238(t *testing.T) {
	db := applyUpToPrev(t)

	// add some software entries
	const insertStmt = `INSERT INTO software
		(name, version, source, bundle_identifier, ` + "`release`" + `, arch, vendor, browser, extension_id)
	VALUES
		(?, ?, ?, ?, ?, ?, ?, ?, ?)`

	sw1 := execNoErrLastID(t, db, insertStmt, "sw1", "1.0", "src1", "", "", "", "", "", "")
	sw2 := execNoErrLastID(t, db, insertStmt, "sw2", "2.0", "src2", "", "", "", "", "", "")
	sw1b := execNoErrLastID(t, db, insertStmt, "sw1", "1.1", "src1", "", "", "", "", "", "")
	sw1c := execNoErrLastID(t, db, insertStmt, "sw1", "1.0", "src1", "bundle1", "", "", "", "", "")
	sw1d := execNoErrLastID(t, db, insertStmt, "sw1", "1.0", "src1", "", "rel1", "", "", "", "")
	sw1e := execNoErrLastID(t, db, insertStmt, "sw1", "1.0", "src1", "", "", "arch1", "", "", "")
	sw1f := execNoErrLastID(t, db, insertStmt, "sw1", "1.0", "src1", "", "", "", "vendor1", "", "")
	sw1g := execNoErrLastID(t, db, insertStmt, "sw1", "1.0", "src1", nil, "", "", "", "browser1", "")
	sw1h := execNoErrLastID(t, db, insertStmt, "sw1", "1.0", "src1", nil, "", "", "", "", "ext1")
	sw1i := execNoErrLastID(t, db, insertStmt, "sw1", "1.0", "src2", "", "", "", "", "", "")

	// Apply current migration.
	applyNext(t, db)

	var swCheck []struct {
		ID       int64  `db:"id"`
		Name     string `db:"name"`
		Checksum string `db:"checksum"`
	}
	err := db.SelectContext(context.Background(), &swCheck, `SELECT id, name, HEX(checksum) AS checksum FROM software ORDER BY id`)
	require.NoError(t, err)
	wantIDs := []int64{sw1, sw2, sw1b, sw1c, sw1d, sw1e, sw1f, sw1g, sw1h, sw1i}
	require.Len(t, swCheck, len(wantIDs))

	gotIDs := make([]int64, len(wantIDs))
	for i, sw := range swCheck {
		if sw.ID == sw2 {
			require.Equal(t, sw.Name, "sw2")
		} else {
			require.Equal(t, sw.Name, "sw1")
		}
		gotIDs[i] = sw.ID
		require.NotEmpty(t, sw.Checksum)
		require.Len(t, sw.Checksum, 32)
	}
	require.Equal(t, wantIDs, gotIDs)
}
