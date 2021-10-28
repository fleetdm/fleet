package scripting

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestRunScript(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   "1",
	})
	require.NoError(t, err)
	require.NotNil(t, host)
	_, err = ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		OsqueryHostID:   "2",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	t.Run("Runs simple select", func(t *testing.T) {
		script := `
main := func() {
	db := import("db");
	res := db.select("select id, node_key from hosts");
	if is_error(res) {
		println(res)
		return
	}
	for host in res {
		printf("host id %d has node key %s\n", int(host.id), host.node_key)
	}
}

main()
`
		expectedOutput := `host id 1 has node key 1
host id 2 has node key 2
`
		output, err := Execute(context.Background(), script, ds.Reader())
		require.NoError(t, err)
		require.Equal(t, expectedOutput, output)
	})

	t.Run("Select accepts parameters", func(t *testing.T) {
		script := `
main := func() {
	db := import("db");
	id := 2
	res := db.select("select id, node_key from hosts where id = ?", id);
	if is_error(res) {
		println(res)
		return
	}
	for host in res {
		printf("host id %d has node key %s\n", int(host.id), host.node_key)
	}
}

main()
`
		expectedOutput := `host id 2 has node key 2
`
		output, err := Execute(context.Background(), script, ds.Reader())
		require.NoError(t, err)
		require.Equal(t, expectedOutput, output)
	})

	t.Run("Select errors out expectedly", func(t *testing.T) {
		script := `
main := func() {
	db := import("db");
	res := db.select("select id, node_key from hosts where id = ?");
	if is_error(res) {
		println(res)
	}
}

main()
`
		expectedOutput := `error: "Error 1064: You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near '?' at line 1"
`
		output, err := Execute(context.Background(), script, ds.Reader())
		require.NoError(t, err)
		require.Equal(t, expectedOutput, output)
	})
}
