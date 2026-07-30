package googleworkspace

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	directory "google.golang.org/api/admin/directory/v1"
)

type fakeAPI struct {
	users     []*directory.User
	groups    []*directory.Group
	members   map[string][]*directory.Member
	usersErr  error
	groupsErr error
	memberErr error
}

func (f *fakeAPI) ListUsers(_ context.Context, _ string) ([]*directory.User, error) {
	return f.users, f.usersErr
}

func (f *fakeAPI) ListGroups(_ context.Context, _ string) ([]*directory.Group, error) {
	return f.groups, f.groupsErr
}

func (f *fakeAPI) ListGroupMembers(_ context.Context, groupKey string) ([]*directory.Member, error) {
	if f.memberErr != nil {
		return nil, f.memberErr
	}
	return f.members[groupKey], nil
}

func TestMapUser(t *testing.T) {
	t.Run("active user with names, department, emails", func(t *testing.T) {
		u := &directory.User{
			Id:           "123",
			PrimaryEmail: "alice@example.com",
			Name:         &directory.UserName{GivenName: "Alice", FamilyName: "Smith"},
			Organizations: []map[string]any{
				{"department": "Sales", "primary": false},
				{"department": "Engineering", "primary": true},
			},
			Emails: []map[string]any{
				{"address": "alice@example.com", "primary": true, "type": "work"},
				{"address": "alice.alt@example.com", "type": "home"},
			},
		}

		su := mapUser(u)
		require.NotNil(t, su.ExternalID)
		assert.Equal(t, "123", *su.ExternalID)
		assert.Equal(t, "alice@example.com", su.UserName)
		require.NotNil(t, su.GivenName)
		assert.Equal(t, "Alice", *su.GivenName)
		require.NotNil(t, su.FamilyName)
		assert.Equal(t, "Smith", *su.FamilyName)
		require.NotNil(t, su.Active)
		assert.True(t, *su.Active)
		require.NotNil(t, su.Department)
		assert.Equal(t, "Engineering", *su.Department, "primary org's department wins")
		require.Len(t, su.Emails, 2)
		assert.Equal(t, "alice@example.com", su.Emails[0].Email)
		require.NotNil(t, su.Emails[0].Primary)
		assert.True(t, *su.Emails[0].Primary)
	})

	t.Run("suspended user is inactive", func(t *testing.T) {
		su := mapUser(&directory.User{Id: "1", PrimaryEmail: "x@example.com", Suspended: true})
		require.NotNil(t, su.Active)
		assert.False(t, *su.Active)
	})

	t.Run("archived user is inactive", func(t *testing.T) {
		su := mapUser(&directory.User{Id: "1", PrimaryEmail: "x@example.com", Archived: true})
		require.NotNil(t, su.Active)
		assert.False(t, *su.Active)
	})

	t.Run("department falls back to first non-empty when no primary org", func(t *testing.T) {
		su := mapUser(&directory.User{
			Id:           "1",
			PrimaryEmail: "x@example.com",
			Organizations: []map[string]any{
				{"department": ""},
				{"department": "Support"},
			},
		})
		require.NotNil(t, su.Department)
		assert.Equal(t, "Support", *su.Department)
	})

	t.Run("primary email synthesized when absent from emails array", func(t *testing.T) {
		su := mapUser(&directory.User{
			Id:           "1",
			PrimaryEmail: "primary@example.com",
			Emails:       []map[string]any{{"address": "other@example.com", "type": "home"}},
		})
		require.Len(t, su.Emails, 2)
		assert.Equal(t, "primary@example.com", su.Emails[0].Email)
		require.NotNil(t, su.Emails[0].Primary)
		assert.True(t, *su.Emails[0].Primary)
	})

	t.Run("duplicate email addresses are de-duplicated", func(t *testing.T) {
		su := mapUser(&directory.User{
			Id:           "1",
			PrimaryEmail: "dup@example.com",
			Emails: []map[string]any{
				{"address": "dup@example.com", "type": "work"},
				{"address": "DUP@example.com", "type": "home"},
			},
		})
		require.Len(t, su.Emails, 1)
		assert.Equal(t, "dup@example.com", su.Emails[0].Email)
	})
}

func TestDirectoryListUsers(t *testing.T) {
	dir := &Directory{
		domain: "example.com",
		api: &fakeAPI{users: []*directory.User{
			{Id: "1", PrimaryEmail: "a@example.com"},
			{Id: "", PrimaryEmail: "noid@example.com"}, // skipped: no ID
			{Id: "2", PrimaryEmail: ""},                // skipped: no email
			{Id: "3", PrimaryEmail: "c@example.com"},
		}},
	}
	users, err := dir.ListUsers(t.Context())
	require.NoError(t, err)
	require.Len(t, users, 2)
	assert.Equal(t, "a@example.com", users[0].UserName)
	assert.Equal(t, "c@example.com", users[1].UserName)
}

func TestDirectoryListUsersError(t *testing.T) {
	dir := &Directory{domain: "example.com", api: &fakeAPI{usersErr: errors.New("boom")}}
	_, err := dir.ListUsers(t.Context())
	require.Error(t, err)
}

func TestDirectoryListGroups(t *testing.T) {
	dir := &Directory{
		domain: "example.com",
		api: &fakeAPI{
			groups: []*directory.Group{
				{Id: "g1", Name: "Engineering", Email: "eng@example.com"},
				{Id: "g2", Email: "ops@example.com"}, // no Name -> display name from email
				{Id: ""},                             // skipped
			},
			members: map[string][]*directory.Member{
				"g1": {
					{Id: "u1", Type: "USER"},
					{Id: "u2"},                // empty type treated as user
					{Id: "g9", Type: "GROUP"}, // nested group skipped
					{Id: ""},                  // skipped
				},
				"g2": {{Id: "u3", Type: "USER"}},
			},
		},
	}
	groups, err := dir.ListGroups(t.Context())
	require.NoError(t, err)
	require.Len(t, groups, 2)

	assert.Equal(t, "g1", groups[0].ExternalID)
	assert.Equal(t, "Engineering", groups[0].DisplayName)
	assert.Equal(t, []string{"u1", "u2"}, groups[0].MemberExternalIDs)

	assert.Equal(t, "ops@example.com", groups[1].DisplayName, "falls back to email when no name")
	assert.Equal(t, []string{"u3"}, groups[1].MemberExternalIDs)
}

func TestDirectoryListGroupsMemberError(t *testing.T) {
	dir := &Directory{
		domain: "example.com",
		api: &fakeAPI{
			groups:    []*directory.Group{{Id: "g1", Name: "Eng"}},
			memberErr: errors.New("members boom"),
		},
	}
	_, err := dir.ListGroups(t.Context())
	require.Error(t, err)
}
