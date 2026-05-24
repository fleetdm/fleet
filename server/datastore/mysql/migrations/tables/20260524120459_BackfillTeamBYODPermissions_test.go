package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260524120459(t *testing.T) {
	db := applyUpToPrev(t)

	// Pre-existing team without BYOD keys in its mdm config — should be backfilled to true.
	execNoErr(t, db, `INSERT INTO teams (name, description, config) VALUES (?, ?, ?)`,
		"team-without-byod", "", `{"mdm": {"enable_disk_encryption": true}}`)
	// Pre-existing team with allow_byod_wipe already false — should be preserved.
	execNoErr(t, db, `INSERT INTO teams (name, description, config) VALUES (?, ?, ?)`,
		"team-explicit-false", "", `{"mdm": {"allow_byod_wipe": false, "allow_byod_lock": false}}`)
	// Team with only one BYOD key set — the other should be backfilled.
	execNoErr(t, db, `INSERT INTO teams (name, description, config) VALUES (?, ?, ?)`,
		"team-partial", "", `{"mdm": {"allow_byod_wipe": false}}`)

	applyNext(t, db)

	type result struct {
		AllowWipe *bool `db:"allow_byod_wipe"`
		AllowLock *bool `db:"allow_byod_lock"`
	}
	var r result
	require.NoError(t, db.Get(&r, `
		SELECT JSON_EXTRACT(config, '$.mdm.allow_byod_wipe') AS allow_byod_wipe,
		       JSON_EXTRACT(config, '$.mdm.allow_byod_lock') AS allow_byod_lock
		FROM teams WHERE name = 'team-without-byod'`))
	require.NotNil(t, r.AllowWipe)
	require.True(t, *r.AllowWipe, "team without keys gets allow_byod_wipe=true")
	require.NotNil(t, r.AllowLock)
	require.True(t, *r.AllowLock, "team without keys gets allow_byod_lock=true")

	require.NoError(t, db.Get(&r, `
		SELECT JSON_EXTRACT(config, '$.mdm.allow_byod_wipe') AS allow_byod_wipe,
		       JSON_EXTRACT(config, '$.mdm.allow_byod_lock') AS allow_byod_lock
		FROM teams WHERE name = 'team-explicit-false'`))
	require.NotNil(t, r.AllowWipe)
	require.False(t, *r.AllowWipe, "explicit false preserved")
	require.NotNil(t, r.AllowLock)
	require.False(t, *r.AllowLock, "explicit false preserved")

	require.NoError(t, db.Get(&r, `
		SELECT JSON_EXTRACT(config, '$.mdm.allow_byod_wipe') AS allow_byod_wipe,
		       JSON_EXTRACT(config, '$.mdm.allow_byod_lock') AS allow_byod_lock
		FROM teams WHERE name = 'team-partial'`))
	require.NotNil(t, r.AllowWipe)
	require.False(t, *r.AllowWipe, "explicit false preserved")
	require.NotNil(t, r.AllowLock)
	require.True(t, *r.AllowLock, "missing key backfilled to true")
}
