package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220208144831(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`INSERT INTO software (name, version, source) VALUES ("authconfig", "6.2.8", "rpm_packages")`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO software (name, version, source, bundle_identifier) VALUES ("iTerm.app", "3.4.14", "apps", "com.googlecode.iterm2")`)
	require.NoError(t, err)

	applyNext(t, db)

	// Check migration removes rpm packages.
	row := db.QueryRow(`SELECT COUNT(*) FROM software WHERE source = "rpm_packages"`)
	var count int
	require.NoError(t, row.Scan(&count))
	require.Zero(t, count)
	row = db.QueryRow(`SELECT COUNT(*) FROM software WHERE source = "apps"`)
	require.NoError(t, row.Scan(&count))
	require.Equal(t, 1, count)

	// Check we can INSERT software with the new columns empty.
	_, err = db.Exec(`INSERT INTO software (name, version, source, bundle_identifier) VALUES ("iCloud.app", "1.0", "apps", "com.apple.CloudKit.ShareBear")`)
	require.NoError(t, err)

	// Check we can INSERT software with the new columns set.
	_, err = db.Exec(`INSERT INTO software (name, version, source, ` + "`release`" + `, vendor, arch) VALUES ("authconfig", "6.2.8", "rpm_packages", "30.el7", "CentOS", "x86_64")`)
	require.NoError(t, err)
}
