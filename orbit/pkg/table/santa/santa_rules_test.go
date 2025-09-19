//go:build darwin
// +build darwin

package santa

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRuleTypeFromInt(t *testing.T) {
	tests := []struct {
		in   int
		want ruleType
	}{
		{1000, ruleTypeBinary},
		{2000, ruleTypeCertificate},
		{3000, ruleTypeTeamID},
		{4000, ruleTypeSigningID},
		{5000, ruleTypeCDHash},
		{42, ruleTypeUnknown},
		{-1, ruleTypeUnknown},
	}
	for _, tc := range tests {
		got := getRuleTypeFromInt(tc.in)
		require.Equal(t, tc.want, got, "input=%d", tc.in)
	}
}

func TestGetRuleStateFromInt(t *testing.T) {
	tests := []struct {
		in   int
		want ruleState
	}{
		{1, ruleStateAllowlist},
		{2, ruleStateBlocklist},
		{0, ruleStateUnknown},
		{-1, ruleStateUnknown},
		{999, ruleStateUnknown},
	}
	for _, tc := range tests {
		got := getRuleStateFromInt(tc.in)
		require.Equal(t, tc.want, got, "input=%d", tc.in)
	}
}

func TestGetRuleTypeName(t *testing.T) {
	tests := []struct {
		in   ruleType
		want string
	}{
		{ruleTypeBinary, "Binary"},
		{ruleTypeCertificate, "Certificate"},
		{ruleTypeTeamID, "TeamID"},
		{ruleTypeSigningID, "SigningID"},
		{ruleTypeCDHash, "CDHash"},
		{ruleTypeUnknown, "Unknown"},
	}
	for _, tc := range tests {
		got := getRuleTypeName(tc.in)
		require.Equal(t, tc.want, got)
	}
}

// helper: create a new SQLite DB with schema + WAL mode
func setupTestDB(t *testing.T, dir string) string {
	dbPath := filepath.Join(dir, "santa.db")

	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		PRAGMA journal_mode=WAL;
		CREATE TABLE rules (
			identifier TEXT,
			state      INTEGER,
			type       INTEGER,
			custommsg  TEXT
		);
	`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	return dbPath
}

// helper: insert rows into the test DB
func seedRules(t *testing.T, dbPath string, rows ...[4]any) {
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	tx, err := db.Begin()
	require.NoError(t, err)

	stmt, err := tx.Prepare(`INSERT INTO rules(identifier, state, type, custommsg) VALUES(?,?,?,?)`)
	require.NoError(t, err)
	defer stmt.Close()

	for _, r := range rows {
		_, err := stmt.Exec(r[0], r[1], r[2], r[3])
		require.NoError(t, err)
	}

	require.NoError(t, tx.Commit())
}

func TestCollectSantaRules_Success(t *testing.T) {
	tmp := t.TempDir()
	dbPath := setupTestDB(t, tmp)

	seedRules(t, dbPath,
		[4]any{"abc", int64(1), int64(1000), "hello"},
		[4]any{"def", int64(2), int64(2000), nil}, // NULL custommsg
	)

	got, err := collectSantaRulesFromPath(context.Background(), dbPath, 2000)
	require.NoError(t, err)

	require.Equal(t, []ruleEntry{
		{identifier: "abc", ruleState: ruleStateAllowlist, ruleType: ruleTypeBinary, customMessage: "hello"},
		{identifier: "def", ruleState: ruleStateBlocklist, ruleType: ruleTypeCertificate, customMessage: ""},
	}, got)
}

func TestCollectSantaRules_DBMissing(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nope.db")

	_, err := collectSantaRulesFromPath(context.Background(), dbPath, 100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestCollectSantaRules_ContextCancel(t *testing.T) {
	tmp := t.TempDir()
	dbPath := setupTestDB(t, tmp)
	seedRules(t, dbPath, [4]any{"x", int64(1), int64(1000), "msg"})

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collectSantaRulesFromPath(ctx, dbPath, 2000)
	require.Error(t, err)
	require.True(t,
		strings.Contains(strings.ToLower(err.Error()), "context") ||
			strings.Contains(strings.ToLower(err.Error()), "canceled") ||
			strings.Contains(strings.ToLower(err.Error()), "cancelled"),
		"expected cancellation error, got %v", err,
	)
}

func TestCollectSantaRules_UnknownValuesAndNulls(t *testing.T) {
	tmp := t.TempDir()
	dbPath := setupTestDB(t, tmp)

	// Unknown type/state, NULL identifier skipped
	seedRules(t, dbPath,
		[4]any{nil, int64(1), int64(1000), "skip-me"}, // skipped
		[4]any{"aaa", int64(999), int64(888), nil},    // Unknown
	)

	got, err := collectSantaRulesFromPath(context.Background(), dbPath, 2000)
	require.NoError(t, err)

	require.Equal(t, []ruleEntry{
		{identifier: "aaa", ruleState: ruleStateUnknown, ruleType: ruleTypeUnknown, customMessage: ""},
	}, got)
}
