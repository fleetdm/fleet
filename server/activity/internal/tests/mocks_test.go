package tests

import (
	"context"
	"strings"

	"github.com/fleetdm/fleet/v4/server/activity"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
)

// Mock implementations for dependencies outside the bounded context

type mockAuthorizer struct{}

func (m *mockAuthorizer) Authorize(ctx context.Context, subject platform_authz.AuthzTyper, action platform_authz.Action) error {
	// Mark authorization as checked (like the real authorizer does)
	if authzCtx, ok := authz_ctx.FromContext(ctx); ok {
		authzCtx.SetChecked()
	}
	return nil // Allow all for integration tests
}

type mockUserProvider struct {
	users map[uint]*activity.User
}

func newMockUserProvider() *mockUserProvider {
	return &mockUserProvider{users: make(map[uint]*activity.User)}
}

func (m *mockUserProvider) AddUser(u *activity.User) {
	m.users[u.ID] = u
}

func (m *mockUserProvider) UsersByIDs(ctx context.Context, ids []uint) ([]*activity.User, error) {
	var result []*activity.User
	for _, id := range ids {
		if u, ok := m.users[id]; ok {
			result = append(result, u)
		}
	}
	return result, nil
}

func (m *mockUserProvider) FindUserIDs(ctx context.Context, query string) ([]uint, error) {
	query = strings.ToLower(query)
	var ids []uint
	for _, u := range m.users {
		if strings.Contains(strings.ToLower(u.Name), query) ||
			strings.Contains(strings.ToLower(u.Email), query) {
			ids = append(ids, u.ID)
		}
	}
	return ids, nil
}
