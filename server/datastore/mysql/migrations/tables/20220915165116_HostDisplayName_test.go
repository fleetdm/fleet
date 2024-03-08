package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220915165116(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`
		INSERT INTO hosts (hostname, osquery_host_id) VALUES ('foo.example.com', 'foo');
	`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO hosts (hostname, osquery_host_id, computer_name) VALUES ('bar.example.com', 'bar', 'bar');
	`)
	require.NoError(t, err)

	applyNext(t, db)

	type dn struct {
		HostID      uint64 `db:"host_id"`
		DisplayName string `db:"display_name"`
	}
	var rows []dn
	require.NoError(t, db.Select(&rows, `SELECT * FROM host_display_names`))
	require.ElementsMatch(t, []dn{
		{1, "foo.example.com"},
		{2, "bar"},
	}, rows)
}
