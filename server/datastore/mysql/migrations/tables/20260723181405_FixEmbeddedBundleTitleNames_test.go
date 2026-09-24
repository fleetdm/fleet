package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260723181405(t *testing.T) {
	db := applyUpToPrev(t)

	dataStmts := `
		INSERT INTO fleet_maintained_apps (name, slug, unique_identifier, platform) VALUES
			('Microsoft Visual Studio Code', 'visual-studio-code/darwin', 'com.microsoft.VSCode', 'darwin');

		INSERT INTO software_titles (id, name, source, bundle_identifier) VALUES
			(1, 'AmphetamineLoginHelper', 'apps', 'com.if.Amphetamine'),
			(2, 'Foo',                    'apps', 'com.example.foo'),
			(3, 'Code',                   'apps', 'com.microsoft.VSCode'),
			(4, 'BarBaz',                 'apps', 'com.example.barbaz'),
			(5, 'No Bundle ID App',       'apps', NULL),
			(6, 'libfoo-helper',          'deb_packages', 'com.example.deb');

		INSERT INTO software (id, checksum, name, version, source, bundle_identifier, title_id) VALUES
			-- 1: helper bug
			(1, 'cs_01', 'Amphetamine',            '5.3.2', 'apps',         'com.if.Amphetamine',   1),
			(2, 'cs_02', 'AmphetamineLoginHelper', '5.3.2', 'apps',         'com.if.Amphetamine',   1),
			-- 2: benign version spread
			(3, 'cs_03', 'Foo', '1.0', 'apps', 'com.example.foo', 2),
			(4, 'cs_04', 'Foo', '2.0', 'apps', 'com.example.foo', 2),
			-- 3: FMA precedence
			(5, 'cs_05', 'Code',        '1.85.0', 'apps', 'com.microsoft.VSCode', 3),
			(6, 'cs_06', 'Code Helper', '1.85.0', 'apps', 'com.microsoft.VSCode', 3),
			-- 4: LCP empty -> shortest fallback ("Foo" vs "BarBaz")
			(7, 'cs_07', 'Foo',    '1.0', 'apps', 'com.example.barbaz', 4),
			(8, 'cs_08', 'BarBaz', '1.0', 'apps', 'com.example.barbaz', 4),
			-- 5: NULL bundle_identifier (out of scope)
			(9, 'cs_09', 'Helper One', '1.0', 'apps', '', 5),
			(10,'cs_10', 'Helper Two', '1.0', 'apps', '', 5),
			-- 6: non-apps source (out of scope)
			(11,'cs_11', 'libfoo',        '1.0', 'deb_packages', 'com.example.deb', 6),
			(12,'cs_12', 'libfoo-helper', '1.0', 'deb_packages', 'com.example.deb', 6);
	`
	_, err := db.Exec(dataStmts)
	require.NoError(t, err)

	applyNext(t, db)

	type titleRow struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}
	var got []titleRow
	err = db.Select(&got, `SELECT id, name FROM software_titles ORDER BY id`)
	require.NoError(t, err)
	require.Equal(t, []titleRow{
		{1, "Amphetamine"},                  // LCP of "Amphetamine" + "AmphetamineLoginHelper"
		{2, "Foo"},                          // single distinct sibling name -> unchanged
		{3, "Microsoft Visual Studio Code"}, // FMA canonical name takes precedence
		{4, "Foo"},                          // LCP empty; shortest of "Foo" / "BarBaz"
		{5, "No Bundle ID App"},             // bundle_identifier IS NULL -> untouched
		{6, "libfoo-helper"},                // source != 'apps' -> untouched
	}, got)
}
