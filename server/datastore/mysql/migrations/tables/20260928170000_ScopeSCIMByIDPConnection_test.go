package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260928170000(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.ExecContext(t.Context(), `INSERT INTO scim_users (user_name) VALUES ('ada@example.com')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `INSERT INTO scim_groups (display_name) VALUES ('Engineering')`)
	require.NoError(t, err)

	applyNext(t, db)

	var connectionID, userConnectionID, groupConnectionID int
	err = db.QueryRowContext(t.Context(), `SELECT id FROM idp_connections WHERE name = 'default' AND is_default = 1`).Scan(&connectionID)
	require.NoError(t, err)
	err = db.QueryRowContext(t.Context(), `SELECT idp_connection_id FROM scim_users WHERE user_name = 'ada@example.com'`).Scan(&userConnectionID)
	require.NoError(t, err)
	err = db.QueryRowContext(t.Context(), `SELECT idp_connection_id FROM scim_groups WHERE display_name = 'Engineering'`).Scan(&groupConnectionID)
	require.NoError(t, err)
	require.Equal(t, connectionID, userConnectionID)
	require.Equal(t, connectionID, groupConnectionID)

	_, err = db.ExecContext(t.Context(), `INSERT INTO idp_connections (name, is_default) VALUES ('Entra', 0)`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO scim_users (user_name, idp_connection_id)
		SELECT 'ada@example.com', id FROM idp_connections WHERE name = 'Entra'`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `INSERT INTO scim_users (user_name, idp_connection_id) VALUES ('ada@example.com', ?)`, connectionID)
	require.Error(t, err)
}
