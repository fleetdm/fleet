package osquery_utils

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// TestWindowsEntraJoinUserQuery runs the detail query against SQLite tables shaped
// like osquery's registry and certificates tables, so the thumbprint extraction,
// the join, the ordering and the no-join case are checked without a Windows host.
func TestWindowsEntraJoinUserQuery(t *testing.T) {
	const joinInfo = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\CloudDomainJoin\JoinInfo`
	const entraIssuer = "net + windows + MS-Organization-Access + 82dbaca4-3e81-46ca-9c73-0950c1eaca97"

	type reg struct{ key, name, data string }
	type cert struct {
		sha1, issuer  string
		notValidAfter int64
	}
	run := func(t *testing.T, regs []reg, certs []cert) []string {
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer db.Close()
		_, err = db.Exec(`CREATE TABLE registry (key TEXT, path TEXT, name TEXT, type TEXT, data TEXT);
			CREATE TABLE certificates (sha1 TEXT, subject TEXT, issuer TEXT, not_valid_after INTEGER);`)
		require.NoError(t, err)
		for _, r := range regs {
			_, err = db.Exec(`INSERT INTO registry VALUES (?, ?, ?, 'REG_SZ', ?)`, r.key, r.key+`\`+r.name, r.name, r.data)
			require.NoError(t, err)
		}
		for _, c := range certs {
			_, err = db.Exec(`INSERT INTO certificates VALUES (?, 'device-id', ?, ?)`, c.sha1, c.issuer, c.notValidAfter)
			require.NoError(t, err)
		}
		rows, err := db.Query(windowsEntraJoinUser.Query)
		require.NoError(t, err)
		defer rows.Close()
		var got []string
		for rows.Next() {
			// osquery renders a NULL aggregate as an empty string
			var upn sql.NullString
			require.NoError(t, rows.Scan(&upn))
			got = append(got, upn.String)
		}
		require.NoError(t, rows.Err())
		return got
	}

	t.Run("joined device returns the join user", func(t *testing.T) {
		got := run(t,
			[]reg{{joinInfo + `\9B7310A6E10ACB7182BEF560EDEDA2C7C9055106`, "UserEmail", "user@example.com"}},
			[]cert{{"9B7310A6E10ACB7182BEF560EDEDA2C7C9055106", entraIssuer, 2000000000}})
		require.Equal(t, []string{"user@example.com"}, got)
	})

	t.Run("thumbprint match is case-insensitive", func(t *testing.T) {
		got := run(t,
			[]reg{{joinInfo + `\9B7310A6E10ACB7182BEF560EDEDA2C7C9055106`, "UserEmail", "user@example.com"}},
			[]cert{{"9b7310a6e10acb7182bef560ededa2c7c9055106", entraIssuer, 2000000000}})
		require.Equal(t, []string{"user@example.com"}, got)
	})

	t.Run("newest certificate wins when two join records exist", func(t *testing.T) {
		got := run(t,
			[]reg{
				{joinInfo + `\AAAA`, "UserEmail", "old@example.com"},
				{joinInfo + `\BBBB`, "UserEmail", "new@example.com"},
			},
			[]cert{
				{"AAAA", entraIssuer, 1000000000},
				{"BBBB", entraIssuer, 2000000000},
			})
		require.Equal(t, []string{"new@example.com"}, got)
	})

	t.Run("join record without a matching Entra certificate is ignored", func(t *testing.T) {
		got := run(t,
			[]reg{{joinInfo + `\AAAA`, "UserEmail", "user@example.com"}},
			[]cert{{"BBBB", entraIssuer, 2000000000}, {"AAAA", "CN=Some Other CA", 2000000000}})
		require.Empty(t, got)
	})

	t.Run("registered-only device has a certificate but no join record", func(t *testing.T) {
		got := run(t, nil, []cert{{"AAAA", entraIssuer, 2000000000}})
		require.Empty(t, got)
	})

	t.Run("join record without a user returns an empty user", func(t *testing.T) {
		got := run(t,
			[]reg{
				{joinInfo + `\AAAA`, "UserEmail", ""},
				{joinInfo + `\AAAA`, "TenantId", "tenant"},
			},
			[]cert{{"AAAA", entraIssuer, 2000000000}})
		require.Equal(t, []string{""}, got)
	})

	t.Run("join record with no UserEmail value at all returns an empty user", func(t *testing.T) {
		got := run(t,
			[]reg{{joinInfo + `\AAAA`, "TenantId", "tenant"}},
			[]cert{{"AAAA", entraIssuer, 2000000000}})
		require.Equal(t, []string{""}, got)
	})

	t.Run("newest record without a user hides an older record's user", func(t *testing.T) {
		got := run(t,
			[]reg{
				{joinInfo + `\OLD1`, "UserEmail", "old@example.com"},
				{joinInfo + `\NEW1`, "TenantId", "tenant"},
			},
			[]cert{
				{"OLD1", entraIssuer, 1000000000},
				{"NEW1", entraIssuer, 2000000000},
			})
		require.Equal(t, []string{""}, got)
	})

	t.Run("not joined at all", func(t *testing.T) {
		require.Empty(t, run(t, nil, nil))
	})
}
