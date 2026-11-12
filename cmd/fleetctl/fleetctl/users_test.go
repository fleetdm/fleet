package fleetctl

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserDelete(t *testing.T) {
	opts := &service.TestServerOpts{}
	_, ds := testing_utils.RunServerWithMockedDS(t, opts)

	ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		return &fleet.User{
			ID:    42,
			Name:  "test1",
			Email: "user1@test.com",
		}, nil
	}

	deletedUser := uint(0)

	ds.DeleteUserIfNotLastAdminFunc = func(ctx context.Context, id uint) error {
		deletedUser = id
		return nil
	}
	opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, activity activity_api.ActivityDetails) error {
		assert.Equal(t, fleet.ActivityTypeDeletedUser{}.ActivityName(), activity.ActivityName())
		return nil
	}

	assert.Empty(t, runAppForTest(t, []string{"user", "delete", "--email", "user1@test.com"}))
	assert.Equal(t, uint(42), deletedUser)
}

// TestUserCreateForcePasswordReset tests that the `fleetctl user create` command
// creates a user with the proper "AdminForcePasswordReset" value depending on
// the passed flags (e.g. SSO users shouldn't be required to do password reset on first login).
func TestUserCreateForcePasswordReset(t *testing.T) {
	_, ds := testing_utils.RunServerWithMockedDS(t)

	pwd := test.GoodPassword

	ds.InviteByEmailFunc = func(ctx context.Context, email string) (*fleet.Invite, error) {
		return nil, &notFoundError{}
	}
	// createdUsers tracks users created during tests so Login can find them by email.
	createdUsers := map[string]*fleet.User{}
	ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		if u, ok := createdUsers[email]; ok {
			return u, nil
		}
		return nil, &notFoundError{}
	}
	var apiOnlyUserSessionKey string
	ds.NewSessionFunc = func(ctx context.Context, userID uint, sessionKeySize int) (*fleet.Session, error) {
		key := make([]byte, sessionKeySize)
		_, err := rand.Read(key)
		if err != nil {
			return nil, err
		}
		sessionKey := base64.StdEncoding.EncodeToString(key)
		apiOnlyUserSessionKey = sessionKey
		return &fleet.Session{
			ID:     2,
			UserID: userID,
			Key:    sessionKey,
		}, nil
	}

	for _, tc := range []struct {
		name                            string
		args                            []string
		expectedAdminForcePasswordReset bool
		displaysToken                   bool
		isAPIOnly                       bool
	}{
		{
			name:                            "sso",
			args:                            []string{"--email", "foo@example.com", "--name", "foo", "--sso"},
			expectedAdminForcePasswordReset: false,
		},
		{
			name:                            "api-only",
			args:                            []string{"--name", "bar", "--api-only"},
			expectedAdminForcePasswordReset: false,
			displaysToken:                   true,
			isAPIOnly:                       true,
		},
		{
			// --sso is ignored by the api-only endpoint, so a password-based user
			// is always created and a token is always returned.
			name:                            "api-only-sso",
			args:                            []string{"--email", "baz@example.com", "--name", "baz", "--api-only", "--sso"},
			expectedAdminForcePasswordReset: false,
			displaysToken:                   true,
			isAPIOnly:                       true,
		},
		{
			name:                            "non-sso-non-api-only",
			args:                            []string{"--email", "zoo@example.com", "--password", pwd, "--name", "zoo"},
			expectedAdminForcePasswordReset: true,
		},
	} {
		ds.NewUserFuncInvoked = false
		ds.NewUserFunc = func(ctx context.Context, user *fleet.User) (*fleet.User, error) {
			assert.Equal(t, tc.expectedAdminForcePasswordReset, user.AdminForcedPasswordReset)
			createdUsers[user.Email] = user
			return user, nil
		}

		stdout := runAppForTest(t, append(
			[]string{"user", "create"},
			tc.args...,
		))
		switch {
		case tc.displaysToken:
			require.Equal(t, fmt.Sprintf("Successfully created new user!\nThe API token for your new user is: %s\n", apiOnlyUserSessionKey), stdout)
		case tc.isAPIOnly:
			require.Equal(t, "Successfully created new user!\n", stdout)
		default:
			require.Empty(t, stdout)
		}
		require.True(t, ds.NewUserFuncInvoked)
	}
}
