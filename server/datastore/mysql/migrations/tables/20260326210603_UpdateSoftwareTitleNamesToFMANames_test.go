package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestUp_20260326210603(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert FMAs with canonical names
	dataStmts := `
	  INSERT INTO fleet_maintained_apps (name, slug, unique_identifier, platform) VALUES
	    ('Microsoft Visual Studio Code', 'visual-studio-code/darwin', 'com.microsoft.VSCode', 'darwin'),
	    ('1Password', '1password/darwin', 'com.1password.1password', 'darwin'),
	    ('Windows App', 'windows-app/windows', 'com.windows.app', 'windows');

	  INSERT INTO software_titles (id, name, source, bundle_identifier) VALUES
	    (1, 'Code', 'apps', 'com.microsoft.VSCode'),
	    (2, '1Password 7', 'apps', 'com.1password.1password'),
	    (3, 'Other App', 'apps', 'com.example.other'),
	    (4, 'No Bundle ID App', 'apps', NULL);

	  INSERT INTO software (id, checksum, name, version, source, bundle_identifier, title_id) VALUES
	    (1, 'checksum_01', 'Code', '1.85.0', 'apps', 'com.microsoft.VSCode', 1),
	    (2, 'checksum_02', 'Code', '1.86.0', 'apps', 'com.microsoft.VSCode', 1),
	    (3, 'checksum_03', '1Password 7', '7.10.0', 'apps', 'com.1password.1password', 2),
	    (4, 'checksum_04', 'Other App', '1.0.0', 'apps', 'com.example.other', 3),
	    (5, 'checksum_05', 'No Bundle ID App', '1.0.0', 'apps', '', 4);
	`

	_, err := db.Exec(dataStmts)
	require.NoError(t, err)

	// Apply the migration
	applyNext(t, db)

	// Verify software_titles were updated correctly
	type softwareTitle struct {
		ID               uint   `db:"id"`
		Name             string `db:"name"`
		BundleIdentifier string `db:"bundle_identifier"`
	}

	var titles []softwareTitle
	err = db.Select(&titles, `SELECT id, name, COALESCE(bundle_identifier, '') as bundle_identifier FROM software_titles ORDER BY id`)
	require.NoError(t, err)
	require.ElementsMatch(t, []softwareTitle{
		{1, "Microsoft Visual Studio Code", "com.microsoft.VSCode"}, // Updated to FMA name
		{2, "1Password", "com.1password.1password"},                 // Updated to FMA name
		{3, "Other App", "com.example.other"},                       // No matching FMA, unchanged
		{4, "No Bundle ID App", ""},                                 // No bundle_identifier, unchanged
	}, titles)

	// Verify software entries were updated correctly
	type softwareRow struct {
		ID               uint   `db:"id"`
		Name             string `db:"name"`
		Version          string `db:"version"`
		BundleIdentifier string `db:"bundle_identifier"`
		TitleID          *uint  `db:"title_id"`
	}

	var software []softwareRow
	err = db.Select(&software, `SELECT id, name, version, COALESCE(bundle_identifier, '') as bundle_identifier, title_id FROM software ORDER BY id`)
	require.NoError(t, err)
	require.ElementsMatch(t, []softwareRow{
		{1, "Microsoft Visual Studio Code", "1.85.0", "com.microsoft.VSCode", ptr.Uint(1)}, // Updated to FMA name
		{2, "Microsoft Visual Studio Code", "1.86.0", "com.microsoft.VSCode", ptr.Uint(1)}, // Updated to FMA name
		{3, "1Password", "7.10.0", "com.1password.1password", ptr.Uint(2)},                 // Updated to FMA name
		{4, "Other App", "1.0.0", "com.example.other", ptr.Uint(3)},                        // No matching FMA, unchanged
		{5, "No Bundle ID App", "1.0.0", "", ptr.Uint(4)},                                  // No bundle_identifier, unchanged
	}, software)
}
