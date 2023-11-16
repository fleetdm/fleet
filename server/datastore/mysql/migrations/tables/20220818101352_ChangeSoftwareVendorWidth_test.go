package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220818101352(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`INSERT INTO software (name, version, source, bundle_identifier, vendor, arch)
	VALUES
	('zchunk-libs', '1.2.1', 'rpm_packages', '', 'Fedora Project', 'x86_64'),
	('zchunk-libs', '1.2.1', 'rpm_packages', '', 'Fedora Project II', 'x86_64'),
	('word', '1.2.1', 'rpm_packages', '', 'Fake MS', 'x86_64'),
	('word', '1.2.2', 'rpm_packages', '', 'Fake MS', 'x86_64'),
	('excel', '1.2.1', 'rpm_packages', '', '', 'x86_64')
	`)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// Check all old vendors are still there
	var vendors []string
	err = db.Select(&vendors, `SELECT vendor FROM software`)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"Fedora Project", "Fedora Project II", "Fake MS", "Fake MS", ""}, vendors)

	// Check we can store a longer vendors
	randVendor := `
	oFZTwTV5WxJt02EVHEBcnhLzuJ8wnxKwfbabPWy7yTSiQbabEcAGDVmoXKZEZJLWObGD0cVfYptInHYgKjtDeDsBh2a8669EnyAqyBECXbFjSh`

	_, err = db.Exec(
		`INSERT INTO software (name, version, source, bundle_identifier, vendor, arch) VALUES  ('zchunk-libs', '1.2.1', 'rpm_packages', '', ?, 'x86_64')`,
		randVendor,
	)
	require.NoError(t, err)
}
