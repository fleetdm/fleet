package main

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
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

	expectedAdminForcePasswordReset := false

	ds.NewUserFunc = func(ctx context.Context, user *fleet.User) (*fleet.User, error) {
		assert.Equal(t, expectedAdminForcePasswordReset, user.AdminForcedPasswordReset)
		return user, nil
	}

	assert.Equal(t, "", runAppForTest(t, []string{
		"user", "create",
		"--email", "foo@example.com",
		"--name", "foo",
		"--sso",
	}))
	assert.Equal(t, "", runAppForTest(t, []string{
		"user", "create",
		"--email", "bar@example.com",
		"--password", "p4ssw0rd.",
		"--name", "bar",
		"--api-only",
	}))
	assert.Equal(t, "", runAppForTest(t, []string{
		"user", "create",
		"--email", "bar@example.com",
		"--name", "bar",
		"--api-only",
		"--sso",
	}))

	expectedAdminForcePasswordReset = true
	assert.Equal(t, "", runAppForTest(t, []string{
		"user", "create",
		"--email", "zoo@example.com",
		"--password", "p4ssw0rd.",
		"--name", "zoo",
	}))
}
