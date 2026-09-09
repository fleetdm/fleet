package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260803182251(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a couple of SCIM groups before the migration.
	_, err := db.ExecContext(t.Context(), "INSERT INTO scim_groups (id, display_name) VALUES (1, 'Engineering B')")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), "INSERT INTO scim_groups (id, display_name) VALUES (2, 'Frontend B')")
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// A valid parent -> child edge can be inserted.
	_, err = db.ExecContext(t.Context(), "INSERT INTO scim_group_group (parent_group_id, child_group_id) VALUES (1, 2)")
	require.NoError(t, err)

	// The (parent, child) pair is unique.
	_, err = db.ExecContext(t.Context(), "INSERT INTO scim_group_group (parent_group_id, child_group_id) VALUES (1, 2)")
	require.Error(t, err)

	// Referencing a non-existent group fails the foreign key.
	_, err = db.ExecContext(t.Context(), "INSERT INTO scim_group_group (parent_group_id, child_group_id) VALUES (1, 999)")
	require.Error(t, err)

	// Deleting a group cascades to its edges.
	_, err = db.ExecContext(t.Context(), "DELETE FROM scim_groups WHERE id = 2")
	require.NoError(t, err)

	var count int
	err = db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM scim_group_group WHERE child_group_id = 2").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
