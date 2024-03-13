package tables

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestUp_20240313143039(t *testing.T) {
	db := applyUpToPrev(t)

	dataStmts := `
INSERT INTO users VALUES
	(1,'2023-07-21','2023-07-21',_binary '$2a$12$n6hwsD7OU2bAXX94551DQOBcNNhfsEPS3Y6JEuLDjsLNvry3lgJjy','0fF81xRQIriYzm5fdXouk3V3tRwsZJhV','admin','admin@email.com',0,'','',0,'admin',0),
	(2,'2023-07-21','2023-07-21',_binary '$2a$12$YxPPOd5TOmYhDlH5CfGIfuxBe4GJ78gbwvtxoBHTTw.symxpVcEZS','JPDLcBcv4j1QwIU+rHoRWBt3HVJC8hnf','User 1','user1@email.com',0,'','',0,NULL,0),
	(3,'2023-07-21','2023-07-21',_binary '$2a$12$u3kuHl44jMojsols1NayLu0pPBwZvnWH6j6ZuDk6HsN4r0jgg7BRu','MoWlTEHH9zR7blcJ0l7/1c4EMnkh/dxq','User 2','user2@email.com',0,'','',0,NULL,0);

INSERT INTO nano_commands
  (command_uuid, request_type, command, created_at, updated_at)
VALUES
  ('nano-command-uuid-1', 'nano', '<?xml', '2023-07-21', '2023-07-21'),
  ('nano-command-uuid-2', 'nano', '<?xml', '2023-07-21', '2023-07-21');

INSERT INTO windows_mdm_commands
  (command_uuid, raw_command, target_loc_uri, created_at, updated_at)
VALUES
  ('win-command-uuid-1', '<?xml', '', '2023-07-21', '2023-07-21'),
  ('win-command-uuid-2', '<?xml', '', '2023-07-21', '2023-07-21');

INSERT INTO
    mdm_apple_configuration_profiles
      (profile_uuid, team_id, identifier, name, mobileconfig, checksum, created_at, uploaded_at)
    VALUES
      ('a1', 0, 'TestPayloadIdentifier', 'TestPayloadName', "<?xml version='1.0'", 'foo', '2023-07-21', '2023-07-21'),
      ('a2', 0, 'TestPayloadIdentifier2', 'TestPayloadName2', "<?xml version='1.0'", 'foo', '2023-07-21', '2023-07-21');

INSERT INTO
    mdm_windows_configuration_profiles
      (profile_uuid, team_id, name, syncml,  created_at, uploaded_at)
    VALUES
      ('w1', 0, 'TestName', "<?xml version='1.0'", '2023-07-21', '2023-07-21'),
      ('w2', 0, 'TestName2', "<?xml version='1.0'", '2023-07-21', '2023-07-21');
`

	_, err := db.Exec(dataStmts)
	require.NoError(t, err)

	applyNext(t, db)

	// check the newly created user_info tables
	type userInfo struct {
		ID       uint   `db:"id"`
		UserID   *uint  `db:"user_id"`
		UserName string `db:"user_name"`
	}
	var userInfos []userInfo
	err = db.Select(
		&userInfos,
		`SELECT user_id, user_name, id FROM user_persistent_info`,
	)
	require.NoError(t, err)
	require.ElementsMatch(t, []userInfo{
		{ID: uint(1), UserID: ptr.Uint(1), UserName: "admin"},
		{ID: uint(2), UserID: ptr.Uint(2), UserName: "User 1"},
		{ID: uint(3), UserID: ptr.Uint(3), UserName: "User 2"},
	}, userInfos)

	// deleting an user doesn't delete the user info
	_, err = db.Exec(`DELETE FROM users WHERE name = "User 1"`)
	require.NoError(t, err)

	var info userInfo
	err = db.Get(
		&info,
		`SELECT user_id, user_name, id FROM user_persistent_info WHERE user_name = "User 1"`,
	)
	require.NoError(t, err)
	require.Nil(t, info.UserID)
	require.Equal(t, "User 1", info.UserName)

	// check the other tables for timestamps and references
	expectedDate, err := time.Parse("2006-01-02", "2023-07-21")
	require.NoError(t, err)

	tables := []string{
		"nano_commands", "windows_mdm_commands",
		"mdm_apple_configuration_profiles", "mdm_windows_configuration_profiles",
	}

	type entity struct {
		CreatedAt            time.Time `db:"created_at"`
		UpdatedAt            time.Time `db:"updated_at"`
		UploadedAt           time.Time `db:"uploaded_at"`
		UserPersistentInfoID *uint     `db:"user_persistent_info_id"`
		FleetOwned           *bool     `db:"fleet_owned"`
	}

	for _, table := range tables {
		updatedTimestamp := "updated_at"
		wantUploaded, wantUpdated := time.Time{}, expectedDate
		if strings.Contains(table, "configuration_profile") {
			updatedTimestamp = "uploaded_at"
			wantUploaded, wantUpdated = expectedDate, time.Time{}
		}

		fetchEntities := func() []entity {
			var entities []entity
			err = db.Select(
				&entities,
				fmt.Sprintf(`
			  SELECT user_persistent_info_id, fleet_owned, created_at, %s
		  	  FROM %s`, updatedTimestamp, table),
			)
			return entities
		}

		entities := fetchEntities()
		require.NoError(t, err)
		require.Len(t, entities, 2)

		// timestamps are not modified, and columns have the
		// expected default values.
		for _, entity := range entities {
			require.EqualValues(t, expectedDate, entity.CreatedAt)
			require.EqualValues(t, wantUpdated, entity.UpdatedAt)
			require.EqualValues(t, wantUploaded, entity.UploadedAt)
			require.Nil(t, entity.UserPersistentInfoID)
			require.Nil(t, entity.FleetOwned)
		}

		_, err = db.Exec(fmt.Sprintf("UPDATE %s SET fleet_owned = 1", table))
		require.NoError(t, err)
		entities = fetchEntities()
		for _, entity := range entities {
			require.Nil(t, entity.UserPersistentInfoID)
			require.True(t, *entity.FleetOwned)
		}

		_, err = db.Exec(fmt.Sprintf("UPDATE %s SET fleet_owned = 0, user_persistent_info_id = 1", table))
		require.NoError(t, err)
		entities = fetchEntities()
		for _, entity := range entities {
			require.Equal(t, uint(1), *entity.UserPersistentInfoID)
			require.False(t, *entity.FleetOwned)
		}

		_, err = db.Exec(fmt.Sprintf("UPDATE %s SET user_persistent_info_id = 9", table))
		require.ErrorContains(t, err, "foreign key constraint fails")
	}

}
