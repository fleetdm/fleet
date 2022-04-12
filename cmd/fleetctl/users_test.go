package main

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserDelete(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		return &fleet.User{
			ID:    42,
			Name:  "test1",
			Email: "user1@test.com",
		}, nil
	}

	deletedUser := uint(0)

	ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
		deletedUser = id
		return nil
	}

	assert.Equal(t, "", runAppForTest(t, []string{"user", "delete", "--email", "user1@test.com"}))
	assert.Equal(t, uint(42), deletedUser)
}

type notFoundError struct{}

var _ fleet.NotFoundError = (*notFoundError)(nil)

func (e *notFoundError) IsNotFound() bool {
	return true
}

func (e *notFoundError) Error() string {
	return ""
}

// TestUserCreateForcePasswordReset tests that the `fleetctl user create` command
// creates a user with the proper "AdminForcePasswordReset" value depending on
// the passed flags (e.g. SSO users shouldn't be required to do password reset on first login).
func TestUserCreateForcePasswordReset(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.InviteByEmailFunc = func(ctx context.Context, email string) (*fleet.Invite, error) {
		return nil, &notFoundError{}
	}

	for _, tc := range []struct {
		name                            string
		args                            []string
		expectedAdminForcePasswordReset bool
	}{
		{
			name:                            "sso",
			args:                            []string{"--email", "foo@example.com", "--name", "foo", "--sso"},
			expectedAdminForcePasswordReset: false,
		},
		{
			name:                            "api-only",
			args:                            []string{"--email", "bar@example.com", "--password", "p4ssw0rd.", "--name", "bar", "--api-only"},
			expectedAdminForcePasswordReset: false,
		},
		{
			name:                            "api-only-sso",
			args:                            []string{"--email", "baz@example.com", "--name", "baz", "--api-only", "--sso"},
			expectedAdminForcePasswordReset: false,
		},
		{
			name:                            "non-sso-non-api-only",
			args:                            []string{"--email", "zoo@example.com", "--password", "p4ssw0rd.", "--name", "zoo"},
			expectedAdminForcePasswordReset: true,
		},
	} {
		ds.NewUserFuncInvoked = false
		ds.NewUserFunc = func(ctx context.Context, user *fleet.User) (*fleet.User, error) {
			assert.Equal(t, tc.expectedAdminForcePasswordReset, user.AdminForcedPasswordReset)
			return user, nil
		}

		require.Equal(t, "", runAppForTest(t, append(
			[]string{"user", "create"},
			tc.args...,
		)))
		require.True(t, ds.NewUserFuncInvoked)
	}
}
