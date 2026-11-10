package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261106172155(t *testing.T) {
	db := applyUpToPrev(t)

	insertLabel := func(name, platform string) int64 {
		res, err := db.Exec(`INSERT INTO labels (name, description, query, platform, label_type, label_membership_type) VALUES (?, '', 'select 1;', ?, 0, 0)`,
			name, platform)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}
	ubuntuID := insertLabel("ubuntu label", "ubuntu")
	centosID := insertLabel("centos label", "centos")
	insertLabel("all platforms label", "")
	insertLabel("darwin label", "darwin")
	insertLabel("windows label", "windows")
	insertLabel("linux label", "linux")

	execNoErr(t, db, `INSERT INTO label_membership (label_id, host_id) VALUES (?, 1), (?, 2)`, ubuntuID, centosID)

	applyNext(t, db)

	platforms := map[string]string{}
	var rows []struct {
		Name     string `db:"name"`
		Platform string `db:"platform"`
	}
	require.NoError(t, db.Select(&rows, `SELECT name, platform FROM labels WHERE label_type = 0`))
	for _, r := range rows {
		platforms[r.Name] = r.Platform
	}
	require.Equal(t, map[string]string{
		"ubuntu label":        "linux",
		"centos label":        "linux",
		"all platforms label": "",
		"darwin label":        "darwin",
		"windows label":       "windows",
		"linux label":         "linux",
	}, platforms)

	var memberships int
	require.NoError(t, db.Get(&memberships, `SELECT COUNT(*) FROM label_membership WHERE label_id IN (?, ?)`, ubuntuID, centosID))
	require.Equal(t, 2, memberships)
}
