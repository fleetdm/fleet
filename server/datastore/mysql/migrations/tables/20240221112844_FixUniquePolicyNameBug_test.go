package tables

import (
	"context"
	"crypto/md5" //nolint:gosec // (only used for tests)
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	computeChecksum := func(policy fleet.Policy) string {
		h := md5.New() //nolint:gosec // (only used for tests)
		// Compute the same way as DB does.
		teamStr := ""
		if policy.TeamID != nil {
			teamStr = fmt.Sprint(*policy.TeamID)
		}
		cols := []string{teamStr, policy.Name}
		_, _ = fmt.Fprint(h, strings.Join(cols, "\x00"))
		checksum := h.Sum(nil)
		return hex.EncodeToString(checksum)
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
		if pc.ID == policy1 { //nolint:gocritic // ignore ifelseChain
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
		assert.Equal(t, computeChecksum(fleet.Policy{PolicyData: fleet.PolicyData{Name: pc.Name}}), strings.ToLower(pc.Checksum))
		assert.Len(t, pc.Checksum, 32)
	}
	assert.Equal(t, wantIDs, gotIDs)
}
