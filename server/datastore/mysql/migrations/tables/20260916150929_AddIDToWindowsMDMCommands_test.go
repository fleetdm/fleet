package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260916150929(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db,
		`INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri) VALUES (?, ?, ?)`,
		"zzz-inserted-first", "<Exec/>", "./Device/Vendor/MSFT/Foo",
	)
	execNoErr(t, db,
		`INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri) VALUES (?, ?, ?)`,
		"aaa-inserted-second", "<Exec/>", "./Device/Vendor/MSFT/Bar",
	)

	applyNext(t, db)

	execNoErr(t, db,
		`INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri) VALUES (?, ?, ?)`,
		"000-inserted-third", "<Exec/>", "./Device/Vendor/MSFT/Baz",
	)

	var uuids []string
	err := db.Select(&uuids, `SELECT command_uuid FROM windows_mdm_commands ORDER BY id ASC`)
	require.NoError(t, err)
	// ALGORITHM=COPY backfills existing rows in primary-key order, and a post-migration insert sorts after them
	// even though its uuid sorts first.
	require.Equal(t, []string{"aaa-inserted-second", "zzz-inserted-first", "000-inserted-third"}, uuids)
}
