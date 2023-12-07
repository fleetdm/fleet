package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20231109115838(t *testing.T) {
	db := applyUpToPrev(t)

	//
	// Insert data to test the migration
	//
	// ...

	userEmails := map[uint]string{1: "admin@email.com", 2: "user1@email.com"}

	setupStmts := `
		INSERT INTO users VALUES 
			(1,'2023-11-03 20:32:32','2023-11-03 20:32:32',_binary '$2a$12$n6hwsD7OU2bAXX94551DQOBcNNhfsEPS3Y6JEuLDjsLNvry3lgJjy','0fF81xRQIriYzm5fdXouk3V3tRwsZJhV','admin','admin@email.com',0,'','',0,'admin',0),
			(2,'2023-11-03 20:33:13','2023-11-03 20:35:26',_binary '$2a$12$YxPPOd5TOmYhDlH5CfGIfuxBe4GJ78gbwvtxoBHTTw.symxpVcEZS','JPDLcBcv4j1QwIU+rHoRWBt3HVJC8hnf','User 1','user1@email.com',0,'','',0,NULL,0);
		INSERT INTO activities VALUES
			(1,'2023-11-04 20:32:32',1,'admin','user_logged_in','{"public_ip": "[::1]"}',0),
			(2,'2023-11-03 20:32:32',2,'User 1','user_logged_in','{"public_ip": "[::1]"}',0);
	`

	_, err := db.Exec(setupStmts)
	require.NoError(t, err)
	// Apply current migration.
	applyNext(t, db)

	stmt := `
		SELECT user_id, user_email FROM activities;
	`
	rows, err := db.Query(stmt)
	require.NoError(t, rows.Err())
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var userEmail string
		var id uint
		err := rows.Scan(&id, &userEmail)
		require.NoError(t, err)
		require.Equal(t, userEmails[id], userEmail)
	}
}
