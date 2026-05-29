package endpoint_verification_accounts

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixture: a representative on-disk shape, modeled on a real accounts.json
// from a developer machine (gaiaID keys, device.resourceId, user.email).
func writeAccountsJSON(t *testing.T, dir string, entries accountsFile) string {
	t.Helper()
	root := filepath.Join(dir, "Library", "Application Support", "Google", "Endpoint Verification")
	require.NoError(t, os.MkdirAll(root, 0o755))
	p := filepath.Join(root, "accounts.json")
	b, err := json.Marshal(entries)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(p, b, 0o600))
	return p
}

func TestGenerate_SingleAccount(t *testing.T) {
	homeDir := t.TempDir()
	writeAccountsJSON(t, homeDir, accountsFile{
		"103923165313941692277": {
			Device: deviceEntry{
				ResourceID: "f60acecb-c136-4965-9b1b-ba089f75eede",
				LastSync:   "2026-04-18T07:21:50.912Z",
			},
			User: userEntry{Email: "robbie@example.com"},
		},
	})

	tbl := &Table{
		logger: zerolog.Nop(),
		userLister: func() ([]userHome, error) {
			return []userHome{{uid: "501", username: "robbiet480", homeDir: homeDir}}, nil
		},
	}

	rows, err := tbl.generate(context.TODO(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	r := rows[0]
	assert.Equal(t, "501", r["uid"])
	assert.Equal(t, "robbiet480", r["username"])
	assert.Equal(t, "103923165313941692277", r["gaia_id"])
	assert.Equal(t, "f60acecb-c136-4965-9b1b-ba089f75eede", r["resource_id"])
	assert.Equal(t, "robbie@example.com", r["email"])
	assert.Equal(t, "2026-04-18T07:21:50.912Z", r["last_sync"])
}

func TestGenerate_MultipleAccounts(t *testing.T) {
	homeDir := t.TempDir()
	writeAccountsJSON(t, homeDir, accountsFile{
		"100": {Device: deviceEntry{ResourceID: "r1"}, User: userEntry{Email: "a@example.com"}},
		"200": {Device: deviceEntry{ResourceID: "r2"}, User: userEntry{Email: "b@example.com"}},
		"300": {Device: deviceEntry{ResourceID: "r3"}, User: userEntry{Email: "c@example.com"}},
	})

	tbl := &Table{
		logger: zerolog.Nop(),
		userLister: func() ([]userHome, error) {
			return []userHome{{uid: "501", username: "robbiet480", homeDir: homeDir}}, nil
		},
	}

	rows, err := tbl.generate(context.TODO(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 3, "every gaia_id becomes a row")

	emails := []string{rows[0]["email"], rows[1]["email"], rows[2]["email"]}
	sort.Strings(emails)
	assert.Equal(t, []string{"a@example.com", "b@example.com", "c@example.com"}, emails)
}

func TestGenerate_SkipsRowsMissingFields(t *testing.T) {
	homeDir := t.TempDir()
	writeAccountsJSON(t, homeDir, accountsFile{
		// Missing resourceId — skipped.
		"100": {Device: deviceEntry{}, User: userEntry{Email: "a@example.com"}},
		// Missing email — skipped.
		"200": {Device: deviceEntry{ResourceID: "r2"}, User: userEntry{}},
		// Complete — kept.
		"300": {Device: deviceEntry{ResourceID: "r3"}, User: userEntry{Email: "c@example.com"}},
	})

	tbl := &Table{
		logger: zerolog.Nop(),
		userLister: func() ([]userHome, error) {
			return []userHome{{uid: "501", username: "u", homeDir: homeDir}}, nil
		},
	}

	rows, err := tbl.generate(context.TODO(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "300", rows[0]["gaia_id"])
}

func TestGenerate_MultipleUsersEachWithAccounts(t *testing.T) {
	homeA := t.TempDir()
	homeB := t.TempDir()
	writeAccountsJSON(t, homeA, accountsFile{
		"100": {Device: deviceEntry{ResourceID: "rA"}, User: userEntry{Email: "alice@example.com"}},
	})
	writeAccountsJSON(t, homeB, accountsFile{
		"200": {Device: deviceEntry{ResourceID: "rB"}, User: userEntry{Email: "bob@example.com"}},
	})

	tbl := &Table{
		logger: zerolog.Nop(),
		userLister: func() ([]userHome, error) {
			return []userHome{
				{uid: "501", username: "alice", homeDir: homeA},
				{uid: "502", username: "bob", homeDir: homeB},
			}, nil
		},
	}

	rows, err := tbl.generate(context.TODO(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byUser := map[string]map[string]string{}
	for _, r := range rows {
		byUser[r["username"]] = r
	}
	assert.Equal(t, "rA", byUser["alice"]["resource_id"])
	assert.Equal(t, "alice@example.com", byUser["alice"]["email"])
	assert.Equal(t, "rB", byUser["bob"]["resource_id"])
}

func TestGenerate_NoEVFile_EmptyResult(t *testing.T) {
	homeDir := t.TempDir() // no accounts.json
	tbl := &Table{
		logger: zerolog.Nop(),
		userLister: func() ([]userHome, error) {
			return []userHome{{uid: "501", username: "robbiet480", homeDir: homeDir}}, nil
		},
	}

	rows, err := tbl.generate(context.TODO(), table.QueryContext{})
	require.NoError(t, err)
	assert.Empty(t, rows, "no EV file = no rows (not an error)")
}

func TestGenerate_MalformedJSON_SkipsSilently(t *testing.T) {
	homeDir := t.TempDir()
	root := filepath.Join(homeDir, "Library", "Application Support", "Google", "Endpoint Verification")
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "accounts.json"), []byte("not json"), 0o600))

	tbl := &Table{
		logger: zerolog.Nop(),
		userLister: func() ([]userHome, error) {
			return []userHome{{uid: "501", username: "u", homeDir: homeDir}}, nil
		},
	}

	rows, err := tbl.generate(context.TODO(), table.QueryContext{})
	require.NoError(t, err, "malformed file should not crash")
	assert.Empty(t, rows)
}

func TestGenerate_UserListerError_EmptyNotErrored(t *testing.T) {
	tbl := &Table{
		logger: zerolog.Nop(),
		userLister: func() ([]userHome, error) {
			return nil, assert.AnError
		},
	}

	rows, err := tbl.generate(context.TODO(), table.QueryContext{})
	require.NoError(t, err, "lister failure becomes empty rows (osquery convention)")
	assert.Empty(t, rows)
}
