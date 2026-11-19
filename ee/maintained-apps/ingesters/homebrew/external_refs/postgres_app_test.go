package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/require"
)

func TestPostgresAppBundleIDs(t *testing.T) {
	app := &maintained_apps.FMAManifestApp{
		UniqueIdentifier: "com.postgresapp.Postgres2",
		UninstallScript: `remove_launchctl_service 'com.postgresapp.Postgres3LoginHelper'
quit_application 'com.postgresapp.Postgres3'
trash "$LOGGED_IN_USER" '~/Library/Application Support/Postgres'
trash "$LOGGED_IN_USER" '~/Library/Preferences/com.postgresapp.Postgres3.plist'`,
	}

	got, err := PostgresAppBundleIDs(app)
	require.NoError(t, err)
	require.Equal(t, `remove_launchctl_service 'com.postgresapp.Postgres2LoginHelper'
quit_application 'com.postgresapp.Postgres2'
trash "$LOGGED_IN_USER" '~/Library/Application Support/Postgres'
trash "$LOGGED_IN_USER" '~/Library/Preferences/com.postgresapp.Postgres2.plist'`, got.UninstallScript)
	require.Equal(t, maintained_apps.GetScriptRef(got.UninstallScript), got.UninstallScriptRef)
}
