package tables

import (
	"context"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20241220114903(t *testing.T) {
	db := applyUpToPrev(t)

	myJSON := `{"foo": "bar"}`
	execNoErr(t, db,
		fmt.Sprintf(`INSERT INTO mdm_apple_declarations (declaration_uuid, identifier, name, raw_json, checksum, team_id) VALUES ('A', 'A', 'nameA', '%s', '', 0)`,
			myJSON))

	// Apply current migration.
	applyNext(t, db)

	var res []struct {
		DeclarationUUID string `db:"declaration_uuid"`
		RawJSON         string `db:"raw_json"`
	}
	err := sqlx.SelectContext(context.Background(), db, &res, `SELECT declaration_uuid, raw_json FROM mdm_apple_declarations`)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, myJSON, res[0].RawJSON)
	require.Equal(t, "A", res[0].DeclarationUUID)

	execNoErr(t, db,
		`INSERT INTO mdm_apple_declarations (declaration_uuid, identifier, name, raw_json, checksum, team_id) VALUES ('B', 'B', 'nameB', '$FLEET_SECRET_BOZO', '', 0)`)

}
