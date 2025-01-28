package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240415104633(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, "INSERT INTO labels (name, query, platform) VALUES (?,?,?)", "NOT macOS 14+ (Sonoma+)", "SELECT 1", "windows")

	// Apply current migration.
	//
	// The case where the name already exists could not be tested because
	// applying the next migration fails drastically when the migration returns
	// an error (it calls log.Fatal) and the test cannot continue after the
	// error, but it has been tested manually.
	applyNext(t, db)

	var names []string
	err := db.Select(&names, `SELECT name FROM labels`)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(names), 2)
	require.Contains(t, names, "macOS 14+ (Sonoma+)")
	require.Contains(t, names, "NOT macOS 14+ (Sonoma+)")
}
