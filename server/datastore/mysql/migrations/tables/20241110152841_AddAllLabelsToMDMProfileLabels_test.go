package tables

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20241110152841(t *testing.T) {
	db := applyUpToPrev(t)

	// insert 2 profiles and 2 declarations
	execNoErr(t, db, `INSERT INTO mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig, checksum, profile_uuid) VALUES (0, 'A', 'nameA', '<plist></plist>', '', 'A')`)
	execNoErr(t, db, `INSERT INTO mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig, checksum, profile_uuid) VALUES (0, 'B', 'nameB', '<plist></plist>', '', 'B')`)

	execNoErr(t, db, `INSERT INTO mdm_apple_declarations (declaration_uuid, identifier, name, raw_json, checksum, team_id) VALUES ('C', 'C', 'nameC', '{"foo": "bar"}', '', 0)`)
	execNoErr(t, db, `INSERT INTO mdm_apple_declarations (declaration_uuid, identifier, name, raw_json, checksum, team_id) VALUES ('D', 'D', 'nameD', '{"foo": "bar"}', '', 0)`)

	// insert 2 profile labels associations: 1 that's exclude any and 1 that's include all
	cfgExcludeAnyID := execNoErrLastID(t, db, `INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, exclude) VALUES ('A', 'foo', true)`)
	cfgIncludeAllID := execNoErrLastID(t, db, `INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, exclude) VALUES ('B', 'bar', false)`)

	declExcludeAnyID := execNoErrLastID(t, db, `INSERT INTO mdm_declaration_labels (apple_declaration_uuid, label_name, exclude) VALUES ('C', 'baz', true)`)
	declIncludeAllID := execNoErrLastID(t, db, `INSERT INTO mdm_declaration_labels (apple_declaration_uuid, label_name, exclude) VALUES ('C', 'boo', false)`)

	// Apply current migration.
	applyNext(t, db)

	var cps []struct {
		ID        int64 `db:"id"`
		Exclude   bool  `db:"exclude"`
		AllLabels bool  `db:"require_all"`
	}

	err := sqlx.SelectContext(context.Background(), db, &cps, `SELECT id, exclude, require_all FROM mdm_configuration_profile_labels`)
	require.NoError(t, err)

	for _, c := range cps {
		// the exclude any should be unchanged
		if c.ID == cfgExcludeAnyID {
			require.True(t, c.Exclude)
			require.False(t, c.AllLabels)
		}

		// the include all should have require_all = true
		if c.ID == cfgIncludeAllID {
			require.False(t, c.Exclude)
			require.True(t, c.AllLabels)
		}
	}

	err = sqlx.SelectContext(context.Background(), db, &cps, `SELECT id, exclude, require_all FROM mdm_declaration_labels`)
	require.NoError(t, err)

	for _, c := range cps {
		// the exclude any should be unchanged
		if c.ID == declExcludeAnyID {
			require.True(t, c.Exclude)
			require.False(t, c.AllLabels)
		}

		// the include all should have require_all = true
		if c.ID == declIncludeAllID {
			require.False(t, c.Exclude)
			require.True(t, c.AllLabels)
		}
	}
}
