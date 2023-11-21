package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20220928100158(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`INSERT INTO host_device_auth (host_id, token) VALUES (1, 'abcd')`)
	require.NoError(t, err)

	var before time.Time
	err = db.QueryRow(`SELECT current_timestamp()`).Scan(&before)
	require.NoError(t, err)

	applyNext(t, db)

	assertRow := func(id int, wantTok string, wantTm time.Time) {
		var token string
		var afterCreated, afterUpdated time.Time
		// check the timestamps for the row that existed before the migation
		err = db.QueryRow(`SELECT token, created_at, updated_at FROM host_device_auth WHERE host_id = ?`, id).Scan(&token, &afterCreated, &afterUpdated)
		require.NoError(t, err)

		require.Equal(t, wantTok, token)
		require.WithinDuration(t, wantTm, afterCreated, time.Second)
		require.WithinDuration(t, wantTm, afterUpdated, time.Second)
	}

	assertRow(1, "abcd", before)

	// refresh the database timestamp
	err = db.QueryRow(`SELECT current_timestamp()`).Scan(&before)
	require.NoError(t, err)

	// create a new row with the timestamps columns now created
	_, err = db.Exec(`INSERT INTO host_device_auth (host_id, token) VALUES (2, 'zzzz')`)
	require.NoError(t, err)
	assertRow(2, "zzzz", before)

	// create a new row with explicit timestamps
	tm := time.Now().Add(time.Hour)
	_, err = db.Exec(`INSERT INTO host_device_auth (host_id, token, created_at, updated_at) VALUES (3, 'AAA', ?, ?)`, tm, tm)
	require.NoError(t, err)
	assertRow(3, "AAA", tm)
}
