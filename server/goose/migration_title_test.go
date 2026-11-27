package goose

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateTitle(t *testing.T) {
	for _, title := range []string{
		"Add an index on hosts platform",
		"Rename Windows MDM settings",
		"Add Google conditional access tables",
		"Clear software installer HTTP ETags",
		"Drop the mdm_apple_installers table",
		"Drop unused tables",
		"Encrypt",
	} {
		require.NoError(t, ValidateTitle(title), title)
	}

	for _, title := range []string{
		"",
		"   ",
		" Add an index",
		"Add an index ",
		"add an index",
		"Add an index.",
		"Add An Index On Hosts",
		"Rename Windows Settings",
	} {
		require.Error(t, ValidateTitle(title), title)
	}
}

func TestCreateMigrationTitle(t *testing.T) {
	dir := t.TempDir()
	ts := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)

	_, err := CreateMigration("AddThing", "add thing", "go", dir, ts)
	require.Error(t, err)

	paths, err := CreateMigration("AddThing", `Add the "thing" table`, "go", dir, ts)
	require.NoError(t, err)
	require.Len(t, paths, 2)
	require.Equal(t, filepath.Join(dir, "20261127100000_AddThing.go"), paths[0])

	b, err := os.ReadFile(paths[0])
	require.NoError(t, err)
	require.Contains(t, string(b), `MigrationClient.AddMigration("Add the \"thing\" table", Up_20261127100000, Down_20261127100000)`)
}

func TestAddMigrationRequiresTitle(t *testing.T) {
	c := &Client{}
	require.Panics(t, func() { c.AddMigration("", nil, nil) })
	c.AddMigration("Add a table", nil, nil)
	require.Equal(t, "Add a table", c.Migrations[0].Title)
}
