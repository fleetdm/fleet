package googleworkspace

import (
	"context"

	directory "google.golang.org/api/admin/directory/v1"
)

// mockDirectory holds the in-memory fixtures returned by the mock DirectoryAPI.
// Tests populate it via SetMockDirectory before triggering a sync (the mock is
// selected by setting the service account client_email to MockEmail).
type mockDirectory struct {
	users         []*directory.User
	groups        []*directory.Group
	groupMembers  map[string][]*directory.Member
	configureErr  error
	listUsersErr  error
	listGroupsErr error
}

// currentMockDirectory is the fixture set used by the mock DirectoryAPI. It is
// package-level to mirror the calendar mock; tests set it via SetMockDirectory.
var currentMockDirectory = &mockDirectory{groupMembers: map[string][]*directory.Member{}}

// SetMockDirectory installs the fixtures the mock DirectoryAPI will return.
// groupMembers is keyed by group id.
func SetMockDirectory(users []*directory.User, groups []*directory.Group, groupMembers map[string][]*directory.Member) {
	if groupMembers == nil {
		groupMembers = map[string][]*directory.Member{}
	}
	currentMockDirectory = &mockDirectory{
		users:        users,
		groups:       groups,
		groupMembers: groupMembers,
	}
}

// SetMockDirectoryErrors configures the mock to return the given errors, to
// exercise fetch-failure handling (e.g. that reconciliation does not run).
func SetMockDirectoryErrors(configureErr, listUsersErr, listGroupsErr error) {
	currentMockDirectory.configureErr = configureErr
	currentMockDirectory.listUsersErr = listUsersErr
	currentMockDirectory.listGroupsErr = listGroupsErr
}

// ResetMockDirectory clears all fixtures and errors.
func ResetMockDirectory() {
	currentMockDirectory = &mockDirectory{groupMembers: map[string][]*directory.Member{}}
}

type mockDirectoryAPI struct {
	dir *mockDirectory
}

func newMockDirectoryAPI() *mockDirectoryAPI {
	return &mockDirectoryAPI{dir: currentMockDirectory}
}

func (m *mockDirectoryAPI) Configure(_ context.Context, _, _, _ string) error {
	return m.dir.configureErr
}

// The mock returns all fixtures in a single page (empty next page token), which
// is sufficient to exercise the mapping and reconciliation logic.
func (m *mockDirectoryAPI) ListUsers(_ context.Context, _, _ string) ([]*directory.User, string, error) {
	if m.dir.listUsersErr != nil {
		return nil, "", m.dir.listUsersErr
	}
	return m.dir.users, "", nil
}

func (m *mockDirectoryAPI) ListGroups(_ context.Context, _, _ string) ([]*directory.Group, string, error) {
	if m.dir.listGroupsErr != nil {
		return nil, "", m.dir.listGroupsErr
	}
	return m.dir.groups, "", nil
}

func (m *mockDirectoryAPI) ListGroupMembers(_ context.Context, groupKey, _ string) ([]*directory.Member, string, error) {
	return m.dir.groupMembers[groupKey], "", nil
}
