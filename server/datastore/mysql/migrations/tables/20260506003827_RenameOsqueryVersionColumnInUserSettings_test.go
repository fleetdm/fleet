package tables

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDown_20260506003827(t *testing.T) {
	require.NoError(t, Down_20260506003827(nil))
}

func TestUp_20260506003827(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert users with various hidden_host_columns settings.
	// User 1: has osquery_version hidden (should be renamed to agent)
	execNoErr(t, db, `INSERT INTO users (name, email, password, salt, settings)
		VALUES (?, ?, '', '', ?)`,
		"user1", "user1@example.com",
		`{"hidden_host_columns":["hostname","osquery_version","public_ip"]}`,
	)
	// User 2: does NOT have osquery_version hidden (should be unchanged)
	execNoErr(t, db, `INSERT INTO users (name, email, password, salt, settings)
		VALUES (?, ?, '', '', ?)`,
		"user2", "user2@example.com",
		`{"hidden_host_columns":["hostname","public_ip"]}`,
	)
	// User 3: no settings at all (should be unchanged)
	execNoErr(t, db, `INSERT INTO users (name, email, password, salt, settings)
		VALUES (?, ?, '', '', ?)`,
		"user3", "user3@example.com",
		`{}`,
	)

	// Apply current migration.
	applyNext(t, db)

	// Verify user1: osquery_version replaced with agent
	var settings1 string
	err := db.QueryRow(`SELECT settings FROM users WHERE email = ?`, "user1@example.com").Scan(&settings1)
	require.NoError(t, err)

	var parsed1 struct {
		HiddenHostColumns []string `json:"hidden_host_columns"`
	}
	require.NoError(t, json.Unmarshal([]byte(settings1), &parsed1))
	require.Contains(t, parsed1.HiddenHostColumns, "agent")
	require.NotContains(t, parsed1.HiddenHostColumns, "osquery_version")
	require.Contains(t, parsed1.HiddenHostColumns, "hostname")
	require.Contains(t, parsed1.HiddenHostColumns, "public_ip")

	// Verify user2: unchanged (no osquery_version to rename)
	var settings2 string
	err = db.QueryRow(`SELECT settings FROM users WHERE email = ?`, "user2@example.com").Scan(&settings2)
	require.NoError(t, err)

	var parsed2 struct {
		HiddenHostColumns []string `json:"hidden_host_columns"`
	}
	require.NoError(t, json.Unmarshal([]byte(settings2), &parsed2))
	require.NotContains(t, parsed2.HiddenHostColumns, "agent")
	require.NotContains(t, parsed2.HiddenHostColumns, "osquery_version")
	require.Contains(t, parsed2.HiddenHostColumns, "hostname")

	// Verify user3: unchanged (no hidden_host_columns)
	var settings3 string
	err = db.QueryRow(`SELECT settings FROM users WHERE email = ?`, "user3@example.com").Scan(&settings3)
	require.NoError(t, err)
	require.NotContains(t, settings3, "agent")
	require.NotContains(t, settings3, "osquery_version")
}
