package mysql

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// TestSchemaHasNoSQLHashFunctions is a cheap regression guard (no DB needed):
// schema.sql must never reintroduce a SQL MD5()/SHA1() call, since MySQL 9.6/9.7
// LTS removed those functions and forbid them in generated columns.
func TestSchemaHasNoSQLHashFunctions(t *testing.T) {
	b, err := os.ReadFile("schema.sql")
	require.NoError(t, err)
	schema := string(b)
	// Match the function name with optional whitespace before the paren, case
	// insensitively, so md5(, MD5 (, sha1(, etc. are all caught.
	require.NotRegexp(t, `(?i)md5\s*\(`, schema, "schema.sql must not use the SQL MD5() function (removed in MySQL 9.6/9.7)")
	require.NotRegexp(t, `(?i)sha1\s*\(`, schema, "schema.sql must not use the SQL SHA1() function (removed in MySQL 9.6/9.7)")
}

// TestMD5HelperMatchesSQL is the ground-truth test for the Go md5 helpers: the
// value computed in Go MUST equal what MySQL's MD5() produced, so that hashes
// stored before this change keep comparing equal. MySQL 9.6/9.7 removed MD5(),
// but the test DB (8.0) still has it, which is exactly why this comparison is
// possible and pinned here.
func TestMD5HelperMatchesSQL(t *testing.T) {
	ds := CreateMySQLDS(t)
	requireLegacySQLMD5(t, ds)
	ctx := t.Context()

	cases := []struct {
		name  string
		input []byte
	}{
		{"empty string", []byte("")},
		{"ascii text", []byte("hello world")},
		{"utf8 text", []byte("héllo wörld 日本語 🚀")},
		{"binary blob", []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0x00, 0x80, 0x7f}},
		{"json", []byte(`{"a":1,"b":[true,null,"x"]}`)},
		{"newlines and nul", []byte("line1\nline2\x00line3")},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Hex form bound as UNHEX(?).
			var sqlHex string
			require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &sqlHex, `SELECT MD5(?)`, c.input))
			require.Equal(t, sqlHex, strings.ToLower(md5ChecksumBytes(c.input)))

			// Raw 16-byte form bound directly into BINARY(16).
			var sqlRaw []byte
			require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &sqlRaw, `SELECT UNHEX(MD5(?))`, c.input))
			require.Equal(t, sqlRaw, md5Checksum(c.input))
		})
	}
}

// TestMDMAppleDeclarationTokenMatchesSQL pins the Go declaration token against
// the previous generated-column expression
// unhex(md5(concat(raw_json, ifnull(secrets_updated_at, ”)))), including the
// DATETIME(6) rendering of a non-NULL secrets_updated_at.
func TestMDMAppleDeclarationTokenMatchesSQL(t *testing.T) {
	ds := CreateMySQLDS(t)
	requireLegacySQLMD5(t, ds)
	ctx := t.Context()

	_, err := ds.writer(ctx).ExecContext(ctx, `CREATE TEMPORARY TABLE decl_token_probe (raw_json MEDIUMTEXT CHARACTER SET utf8mb4, secrets_updated_at DATETIME(6) NULL)`)
	require.NoError(t, err)

	ts := time.Date(2026, 3, 15, 12, 34, 56, 123456000, time.UTC)
	tsRounded := time.Date(2025, 12, 31, 23, 59, 59, 999999500, time.UTC) // rounds up to next second
	cases := []struct {
		rawJSON          string
		secretsUpdatedAt *time.Time
	}{
		{`{"Type":"com.apple.configuration.foo","Identifier":"x"}`, nil},
		{`{"Type":"com.apple.configuration.foo","Identifier":"x"}`, &ts},
		{`{"a":1}`, &tsRounded},
	}
	for _, c := range cases {
		_, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM decl_token_probe`)
		require.NoError(t, err)
		_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO decl_token_probe (raw_json, secrets_updated_at) VALUES (?, ?)`, c.rawJSON, c.secretsUpdatedAt)
		require.NoError(t, err)

		var sqlVal []byte
		require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &sqlVal,
			`SELECT UNHEX(MD5(CONCAT(raw_json, IFNULL(secrets_updated_at, '')))) FROM decl_token_probe`))
		d := &fleet.MDMAppleDeclaration{RawJSON: json.RawMessage(c.rawJSON), SecretsUpdatedAt: c.secretsUpdatedAt}
		require.Equal(t, sqlVal, d.ComputeToken())
	}
}

// TestMDMAndroidProfileChecksum verifies the android profile checksum is computed
// in Go (no DB round-trip) over a canonical JSON form: it is stable across
// semantically-identical inputs that differ only by key order or whitespace, and
// it changes when the content changes.
func TestMDMAndroidProfileChecksum(t *testing.T) {
	// Same content, different key order + whitespace → same checksum.
	a, err := md5ChecksumFromJSON(json.RawMessage(`{"b":1,"a":2,"nested":{"y":2,"x":1}}`))
	require.NoError(t, err)
	b, err := md5ChecksumFromJSON(json.RawMessage("{\n  \"nested\": { \"x\": 1, \"y\": 2 },\n  \"a\": 2,\n  \"b\": 1\n}"))
	require.NoError(t, err)
	require.Equal(t, a, b, "checksum must be independent of key order and whitespace")

	// Different content → different checksum.
	c, err := md5ChecksumFromJSON(json.RawMessage(`{"a":2,"b":2,"nested":{"x":1,"y":2}}`))
	require.NoError(t, err)
	require.NotEqual(t, a, c)

	// Array order is significant → different checksum.
	d1, err := md5ChecksumFromJSON(json.RawMessage(`{"list":[1,2,3]}`))
	require.NoError(t, err)
	d2, err := md5ChecksumFromJSON(json.RawMessage(`{"list":[3,2,1]}`))
	require.NoError(t, err)
	require.NotEqual(t, d1, d2)

	// Invalid JSON is reported as an error.
	_, err = md5ChecksumFromJSON(json.RawMessage(`{not json`))
	require.Error(t, err)
}
