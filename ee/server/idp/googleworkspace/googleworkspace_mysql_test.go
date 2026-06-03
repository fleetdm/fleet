package googleworkspace

import (
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	directory "google.golang.org/api/admin/directory/v1"
)

// TestSyncIntegration exercises Sync against a real MySQL datastore so the SCIM
// upserts, group membership, and reconcile-deletes are validated end to end
// (including foreign-key cascade behavior). Requires MYSQL_TEST=1.
func TestSyncIntegration(t *testing.T) {
	t.Cleanup(ResetMockDirectory)
	ds := mysqltest.CreateMySQLDS(t)
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)

	// First sync: two users, one group containing the first user.
	SetMockDirectory(
		[]*directory.User{
			gwUser("g1", "alice@example.com", "Alice", "Smith", "Engineering", false),
			gwUser("g2", "bob@example.com", "Bob", "Jones", "Sales", false),
		},
		[]*directory.Group{{Id: "grp1", Name: "Engineers", Email: "eng@example.com"}},
		map[string][]*directory.Member{"grp1": {{Id: "g1", Email: "alice@example.com", Type: "USER"}}},
	)
	require.NoError(t, Sync(ctx, ds, mockIntegration(), logger))

	_, total, err := ds.ListScimUsers(ctx, fleet.ScimUsersListOptions{ScimListOptions: fleet.ScimListOptions{StartIndex: 1, PerPage: 100}})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)

	alice, err := ds.ScimUserByUserName(ctx, "alice@example.com")
	require.NoError(t, err)
	require.NotNil(t, alice.Department)
	assert.Equal(t, "Engineering", *alice.Department)
	require.Len(t, alice.Groups, 1)
	assert.Equal(t, "Engineers", alice.Groups[0].DisplayName)

	// Second sync: bob is gone, group renamed and now empty. Reconcile should
	// delete bob and the old group, and the membership/email rows cascade.
	SetMockDirectory(
		[]*directory.User{gwUser("g1", "alice@example.com", "Alice", "Smith", "Platform", false)},
		[]*directory.Group{{Id: "grp1", Name: "Engineers", Email: "eng@example.com"}},
		map[string][]*directory.Member{"grp1": {{Id: "g1", Type: "USER"}}},
	)
	require.NoError(t, Sync(ctx, ds, mockIntegration(), logger))

	_, total, err = ds.ListScimUsers(ctx, fleet.ScimUsersListOptions{ScimListOptions: fleet.ScimListOptions{StartIndex: 1, PerPage: 100}})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total, "bob should be reconciled away")

	_, err = ds.ScimUserByUserName(ctx, "bob@example.com")
	assert.True(t, fleet.IsNotFound(err))

	alice, err = ds.ScimUserByUserName(ctx, "alice@example.com")
	require.NoError(t, err)
	require.NotNil(t, alice.Department)
	assert.Equal(t, "Platform", *alice.Department, "department should be updated on re-sync")
}
