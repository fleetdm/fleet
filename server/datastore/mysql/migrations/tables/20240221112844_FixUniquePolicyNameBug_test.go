package tables

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUp_20240221112844(t *testing.T) {
	db := applyUpToPrev(t)

	checksumCol := func() string {
		// concatenate with separator \x00
		return ` UNHEX(
		MD5(
			CONCAT_WS(CHAR(0),
				COALESCE(team_id, ''),
				name
			)
		)
	) `
	}

	// Insert 3 policies with the same name but different checksums (which is the bug)
	policy1 := execNoErrLastID(
		t, db, fmt.Sprintf("INSERT INTO policies (name, query, description, checksum) VALUES (?,?,?,%s)", checksumCol()), "policy", "", "",
	)
	policy2 := execNoErrLastID(
		t, db, "INSERT INTO policies (name, query, description, checksum) VALUES (?,?,?,?)", "policy", "", "", "checksum",
	)
	policy3 := execNoErrLastID(
		t, db, "INSERT INTO policies (name, query, description, checksum) VALUES (?,?,?,?)", "policy", "", "", "checksum2",
	)
	// Insert another policy with the name that one of the above policies will be attempted to be renamed into.
	policy4 := execNoErrLastID(
		t, db, fmt.Sprintf("INSERT INTO policies (name, query, description, checksum) VALUES (?,?,?,%s)", checksumCol()), "policy2", "", "",
	)

	// Apply current migration.
	applyNext(t, db)

	var policyCheck []struct {
		ID       int64  `db:"id"`
		Name     string `db:"name"`
		Checksum string `db:"checksum"`
	}
	err := db.SelectContext(context.Background(), &policyCheck, `SELECT id, name, HEX(checksum) AS checksum FROM policies ORDER BY id`)
	require.NoError(t, err)
	wantIDs := []int64{policy1, policy2, policy3, policy4}
	assert.Len(t, policyCheck, len(wantIDs))

	gotIDs := make([]int64, len(wantIDs))
	for i, pc := range policyCheck {
		if pc.ID == policy1 {
			assert.Equal(t, "policy", pc.Name)
		} else if pc.ID == policy2 {
			assert.Equal(t, "policy3", pc.Name) // name changed
			assert.NotEqual(t, "checksum", pc.Checksum)
		} else if pc.ID == policy3 {
			assert.Equal(t, "policy4", pc.Name) // name changed
			assert.NotEqual(t, "checksum2", pc.Checksum)
		} else { // policy4
			assert.Equal(t, "policy2", pc.Name) // name was not changed
		}
		gotIDs[i] = pc.ID
		assert.NotEmpty(t, pc.Checksum)
		assert.Len(t, pc.Checksum, 32)
	}
	assert.Equal(t, wantIDs, gotIDs)
}
