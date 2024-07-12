package tables

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20240709175341(t *testing.T) {
	db := applyUpToPrev(t)
	then := time.Now().UTC().Add(-time.Hour).Round(time.Second)

	for i := 0; i < 4; i++ {
		// create some hosts with uuids that will be used in migration to populate new columns in mdm_idp_accounts
		id := execNoErrLastID(t, db, `INSERT INTO hosts (uuid) VALUES (?);`,
			fmt.Sprintf("host_uuid%d", i),
		)

		// insert host_mdm records that will be used in migration to populate new columns in mdm_idp_accounts
		execNoErr(t, db, `INSERT INTO host_mdm (host_id, fleet_enroll_ref) VALUES (?, ?);`,
			id, fmt.Sprintf("uuid%d", i),
		)

		execNoErr(t, db, `
INSERT INTO 
	mdm_idp_accounts (uuid, username, fullname, email, created_at, updated_at)
VALUES 
	(?,?,?,?,?,?)
`, fmt.Sprintf("uuid%d", i), fmt.Sprintf("username%d", i), fmt.Sprintf("fullname%d", i), fmt.Sprintf("email%d", i), then, then)
	}

	// insert an orphaned mdm_idp_account
	execNoErr(t, db, `
INSERT INTO 
	mdm_idp_accounts (uuid, username, fullname, email, created_at, updated_at)
VALUES 
	(?,?,?,?,?,?)`, "uuid4", "username4", "fullname4", "email4", then, then)

	// Apply current migration.
	applyNext(t, db)

	var dest []struct {
		fleet.MDMIdPAccount
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	require.NoError(t, sqlx.SelectContext(context.Background(), db, &dest, `SELECT * FROM mdm_idp_accounts ORDER BY uuid;`))
	require.Len(t, dest, 5)

	for i, got := range dest {
		require.Equal(t, fmt.Sprintf("uuid%d", i), got.UUID)         // no change
		require.Equal(t, fmt.Sprintf("username%d", i), got.Username) // no change
		require.Equal(t, fmt.Sprintf("fullname%d", i), got.Fullname) // no change
		require.Equal(t, fmt.Sprintf("email%d", i), got.Email)       // no change
		require.Equal(t, then, got.CreatedAt)                        // no change
		require.Equal(t, then, got.UpdatedAt)                        // no change

		if i == 4 {
			// this is the orphaned mdm_idp_account
			require.Equal(t, "uuid4", got.FleetEnrollRef) // new column fleet_enroll_ref is set to uuid in migration
			require.Empty(t, got.HostUUID)                // new column host_uuid is not set in migration because there is no matching host
		} else {
			require.Equal(t, fmt.Sprintf("host_uuid%d", i), got.HostUUID)  // new column host_uuid is set to host uuid in migration
			require.Equal(t, fmt.Sprintf("uuid%d", i), got.FleetEnrollRef) // new column fleet_enroll_ref is set to uuid in migration
		}

	}
}
