package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261113153509(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier)
		VALUES ('Microsoft Teams', 'microsoft-teams/windows', 'windows', 'Microsoft Teams')`)

	applyNext(t, db)

	var exact bool
	require.NoError(t, db.Get(&exact, `SELECT exact_name_match FROM fleet_maintained_apps WHERE slug = 'microsoft-teams/windows'`))
	require.False(t, exact, "existing apps keep prefix matching until the apps list sync sets the flag")
}
