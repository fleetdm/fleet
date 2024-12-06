package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"

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
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		assert.Equal(t, fleet.ActivityTypeDeletedUser{}.ActivityName(), activity.ActivityName())
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

	pwd := test.GoodPassword

	ds.InviteByEmailFunc = func(ctx context.Context, email string) (*fleet.Invite, error) {
		return nil, &notFoundError{}
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		if email == "bar@example.com" {
			apiOnlyUser := &fleet.User{
				ID:    1,
				Email: email,
			}
			err := apiOnlyUser.SetPassword(pwd, 24, 10)
			require.NoError(t, err)
			return apiOnlyUser, nil
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
	}{
		{
			name:                            "sso",
			args:                            []string{"--email", "foo@example.com", "--name", "foo", "--sso"},
			expectedAdminForcePasswordReset: false,
			displaysToken:                   false,
		},
		{
			name:                            "api-only",
			args:                            []string{"--email", "bar@example.com", "--password", pwd, "--name", "bar", "--api-only"},
			expectedAdminForcePasswordReset: false,
			displaysToken:                   true,
		},
		{
			name:                            "api-only-sso",
			args:                            []string{"--email", "baz@example.com", "--name", "baz", "--api-only", "--sso"},
			expectedAdminForcePasswordReset: false,
			displaysToken:                   false,
		},
		{
			name:                            "non-sso-non-api-only",
			args:                            []string{"--email", "zoo@example.com", "--password", pwd, "--name", "zoo"},
			expectedAdminForcePasswordReset: true,
			displaysToken:                   false,
		},
	} {
		ds.NewUserFuncInvoked = false
		ds.NewUserFunc = func(ctx context.Context, user *fleet.User) (*fleet.User, error) {
			assert.Equal(t, tc.expectedAdminForcePasswordReset, user.AdminForcedPasswordReset)
			return user, nil
		}

		stdout := runAppForTest(t, append(
			[]string{"user", "create"},
			tc.args...,
		))
		if tc.displaysToken {
			require.Equal(t, stdout, fmt.Sprintf("Success! The API token for your new user is: %s\n", apiOnlyUserSessionKey))
		} else {
			require.Empty(t, stdout)
		}
		require.True(t, ds.NewUserFuncInvoked)
	}
}

func writeTmpCsv(t *testing.T, contents string) string {
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.csv")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(contents)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())
	return tmpFile.Name()
}

func TestCreateBulkUsers(t *testing.T) {
	_, ds := runServerWithMockedDS(t)
	ds.InviteByEmailFunc = func(ctx context.Context, email string) (*fleet.Invite, error) {
		return nil, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.TeamsSummaryFunc = func(ctx context.Context) ([]*fleet.TeamSummary, error) {
		team1 := &fleet.TeamSummary{
			ID: 1,
		}
		team2 := &fleet.TeamSummary{
			ID: 2,
		}
		return []*fleet.TeamSummary{team1, team2}, nil
	}

	csvFile := writeTmpCsv(t,
		`Name,Email,SSO,API Only,Global Role,Teams
		user11,user11@example.com,false,false,maintainer,
		user12,user12@example.com,false,false,,
		user13,user13@example.com,true,false,admin,
		user14,user14@example.com,false,false,,2:maintainer
		user15,user15@example.com,false,false,,1:admin
		user16,user16@example.com,false,false,,1:admin 2:maintainer`)

	expectedText := `{"kind":"user_roles","apiVersion":"v1","spec":{"roles":{"admin1@example.com":{"global_role":"admin","teams":null},"user11@example.com":{"global_role":"maintainer","teams":null},"user12@example.com":{"global_role":"observer","teams":null},"user13@example.com":{"global_role":"admin","teams":null},"user14@example.com":{"global_role":null,"teams":[{"team":"","role":"maintainer"}]},"user15@example.com":{"global_role":null,"teams":[{"team":"","role":"admin"}]},"user16@example.com":{"global_role":null,"teams":[{"team":"","role":"admin"},{"team":"","role":"maintainer"}]},"user1@example.com":{"global_role":"observer","teams":null},"user2@example.com":{"global_role":"observer","teams":null}}}}
`

	assert.Equal(t, "", runAppForTest(t, []string{"user", "create-users", "--csv", csvFile}))
	assert.Equal(t, expectedText, runAppForTest(t, []string{"get", "user_roles", "--json"}))
}

func TestDeleteBulkUsers(t *testing.T) {
	_, ds := runServerWithMockedDS(t)
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	csvFilePath := writeTmpCsv(t,
		`Email
	user11@example.com
	user12@example.com
	user13@example.com`)

	csvFile, err := os.Open(csvFilePath)
	require.NoError(t, err)
	defer csvFile.Close()

	csvLines, err := csv.NewReader(csvFile).ReadAll()
	require.NoError(t, err)

	users := []fleet.User{}
	deletedUserIds := []uint{}
	for _, user := range csvLines[1:] {
		email := user[0]
		name := strings.Split(email, "@")[0]

		randId, err := rand.Int(rand.Reader, big.NewInt(1000))
		require.NoError(t, err)
		id := uint(randId.Int64()) //nolint:gosec // dismiss G115

		users = append(users, fleet.User{
			Name:  name,
			Email: email,
			ID:    id,
		})
		deletedUserIds = append(deletedUserIds, id)
	}

	for _, user := range users {
		ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return &user, nil
		}
	}
	deletedUser := uint(0)

	ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
		deletedUser = id
		return nil
	}

	assert.Equal(t, "", runAppForTest(t, []string{"user", "delete-users", "--csv", csvFilePath}))
	for indx, user := range users {
		deletedUser = deletedUserIds[indx]
		assert.Equal(t, user.ID, deletedUser)
	}
}
