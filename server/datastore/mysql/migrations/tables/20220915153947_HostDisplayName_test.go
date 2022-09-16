package tables

import (
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"testing"
)

type dn struct {
	HostID      uint64 `db:"host_id"`
	DisplayName string `db:"display_name"`
}

func TestUp_20220915153947(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`
		INSERT INTO hosts (hostname, osquery_host_id) VALUES ('foo.example.com', 'foobar');
	`)
	require.NoError(t, err)

	applyNext(t, db)

	_, err = db.Exec(`
		INSERT INTO hosts (hostname, osquery_host_id) VALUES ('bar.example.com', 'foobaz');
	`)
	require.NoError(t, err)
	checkHDN(t, db, []dn{
		{1, "foo.example.com"},
		{2, "bar.example.com"},
	})
	_, err = db.Exec(`
		UPDATE hosts SET hostname='baz.example.com' WHERE id=1;
	`)
	require.NoError(t, err)
	checkHDN(t, db, []dn{
		{1, "baz.example.com"},
		{2, "bar.example.com"},
	})
	_, err = db.Exec(`
		UPDATE hosts SET computer_name='atari' WHERE id=2;
	`)
	require.NoError(t, err)
	checkHDN(t, db, []dn{
		{1, "baz.example.com"},
		{2, "atari"},
	})
	_, err = db.Exec(`
		UPDATE hosts SET hostname='atari.example.com' WHERE id=2;
	`)
	require.NoError(t, err)
	checkHDN(t, db, []dn{
		{1, "baz.example.com"},
		{2, "atari"},
	})
	_, err = db.Exec(`
		DELETE FROM hosts WHERE id=2;
	`)
	require.NoError(t, err)
	checkHDN(t, db, []dn{
		{1, "baz.example.com"},
	})
}

func checkHDN(t *testing.T, db *sqlx.DB, expect []dn) {
	q, err := db.Queryx(`SELECT * FROM hosts_display_name`)
	require.NoError(t, err)
	t.Cleanup(func() { q.Close() })
	var rows []dn
	for q.Next() {
		var row dn
		require.NoError(t, q.StructScan(&row))
		rows = append(rows, row)
	}
	require.NoError(t, q.Err())
	require.ElementsMatch(t, expect, rows)
}
