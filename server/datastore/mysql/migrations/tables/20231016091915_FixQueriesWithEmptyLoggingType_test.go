package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20231016091915(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := `INSERT INTO queries (
		name, description, query, logging_type
	) VALUES (?, ?, ?, ?), (?, ?, ?, ?), (?, ?, ?, ?)`

	_, err := db.Exec(insertStmt,
		"foobar", "logging_type set to something else", "SELECT 1;", fleet.LoggingDifferential,
		"zoobar", "logging_type unset", "SELECT 2;", "",
		"boobar", "logging_type set to snapshot already", "SELECT 3;", fleet.LoggingSnapshot,
	)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	var foobarLogging string
	err = db.Get(&foobarLogging, "SELECT logging_type FROM queries WHERE name = ?", "foobar")
	require.NoError(t, err)
	require.Equal(t, fleet.LoggingDifferential, foobarLogging)
	var zoobarLogging string
	err = db.Get(&zoobarLogging, "SELECT logging_type FROM queries WHERE name = ?", "zoobar")
	require.NoError(t, err)
	require.Equal(t, fleet.LoggingSnapshot, zoobarLogging)
	var boobarLogging string
	err = db.Get(&boobarLogging, "SELECT logging_type FROM queries WHERE name = ?", "boobar")
	require.NoError(t, err)
	require.Equal(t, fleet.LoggingSnapshot, boobarLogging)
}
