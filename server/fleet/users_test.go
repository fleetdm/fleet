package fleet

import (
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestIsGlobalObserver(t *testing.T) {
	testCases := []struct {
		GlobalRole *string
		Expected   bool
	}{
		{
			GlobalRole: nil,
		},
		{
			GlobalRole: ptr.String(RoleAdmin),
		},
		{
			GlobalRole: ptr.String(RoleObserver),
			Expected:   true,
		},
		{
			GlobalRole: ptr.String(RoleObserverPlus),
			Expected:   true,
		},
	}

	for _, tC := range testCases {
		sut := User{GlobalRole: tC.GlobalRole}
		require.Equal(t, sut.IsGlobalObserver(), tC.Expected)
	}
}

func TestTeamMembership(t *testing.T) {
	teams := []UserTeam{
		{
			Role: RoleAdmin,
			Team: Team{
				ID: 1,
			},
		},
		{
			Role: RoleGitOps,
			Team: Team{
				ID: 2,
			},
		},
		{
			Role: RoleObserver,
			Team: Team{
				ID: 3,
			},
		},
		{
			Role: RoleObserver,
			Team: Team{
				ID: 4,
			},
		},
	}

	sut := User{}
	require.Empty(t, sut.TeamMembership(func(ut UserTeam) bool {
		return true
	}))

	sut.Teams = teams

	var result []uint
	pred := func(ut UserTeam) bool {
		return ut.Role == RoleGitOps || ut.Role == RoleObserver
	}
	for k := range sut.TeamMembership(pred) {
		result = append(result, k)
	}
	require.ElementsMatch(t, result, []uint{2, 3, 4})

	result = make([]uint, 0, len(teams))
	pred = func(ut UserTeam) bool {
		return true
	}
	for k := range sut.TeamMembership(pred) {
		result = append(result, k)
	}
	require.ElementsMatch(t, result, []uint{1, 2, 3, 4})
}

func TestValidatePassword(t *testing.T) {
	passwordTests := []struct {
		Password, Email      string
		Admin, PasswordReset bool
	}{
		{"foobar", "mike@fleet.co", true, false},
		{"bar0baz!?", "jason@fleet.co", true, false},
	}

	for _, tt := range passwordTests {
		user := newTestUser(t, tt.Password, tt.Email)

		err := user.ValidatePassword(tt.Password)
		assert.Nil(t, err)

		err = user.ValidatePassword("different")
		assert.NotNil(t, err)
	}
}

func newTestUser(t *testing.T, password, email string) *User {
	var (
		salt = "test-salt"
		cost = 10
	)
	withSalt := []byte(fmt.Sprintf("%s%s", password, salt))
	hashed, _ := bcrypt.GenerateFromPassword(withSalt, cost)
	return &User{
		Salt:     salt,
		Password: hashed,
		Email:    email,
	}
}

func TestUserPasswordRequirements(t *testing.T) {
	passwordTests := []struct {
		password string
		wantErr  bool
	}{
		{
			password: "foobar",
			wantErr:  true,
		},
		{
			password: "foobarbaz",
			wantErr:  true,
		},
		{
			password: "foobarbaz!",
			wantErr:  true,
		},
		{
			password: "foobarbaz!3",
			wantErr:  true,
		},
		{
			password: "foobarbaz!3!",
		},
	}

	for _, tt := range passwordTests {
		t.Run(tt.password, func(t *testing.T) {
			err := ValidatePasswordRequirements(tt.password)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestSaltAndHashPassword(t *testing.T) {
	goodTests := []string{"foobar!!", "bazbing!!", "foobarbaz!!!foobarbaz!!!foobarbaz!!!foobarbaz!!", "foobarbaz!!!foobarbaz!!!foobarbaz!!!foobarbaz!!!"}
	keySize := 24
	cost := 10

	for _, pwd := range goodTests {
		hashed, salt, err := saltAndHashPassword(keySize, pwd, cost)
		require.NoError(t, err)

		saltAndPass := []byte(fmt.Sprintf("%s%s", pwd, salt))
		err = bcrypt.CompareHashAndPassword(hashed, saltAndPass)
		require.NoError(t, err)

		err = bcrypt.CompareHashAndPassword(hashed, []byte(fmt.Sprint("invalidpassword", salt)))
		require.Error(t, err)

		// too long
		badTests := []string{"foobarbaz!!!foobarbaz!!!foobarbaz!!!foobarbaz!!!!"}
		for _, pwd := range badTests {
			_, _, err := saltAndHashPassword(keySize, pwd, cost)
			require.Error(t, err)

		}
	}
}

func TestAdminCreateValidate(t *testing.T) {
	testCases := []struct {
		payload UserPayload
		// if errContains is empty then no error is expected
		errContains []string
	}{
		{
			payload:     UserPayload{},
			errContains: []string{"name", "email", "password"},
		},
		{
			payload:     UserPayload{Name: ptr.String(""), Email: ptr.String(""), Password: ptr.String("")},
			errContains: []string{"name", "email", "password"},
		},
		{
			payload:     UserPayload{Name: ptr.String("Foo"), Email: ptr.String(""), Password: ptr.String("")},
			errContains: []string{"email", "password"},
		},
		{
			payload:     UserPayload{Name: ptr.String("Foo"), Email: ptr.String("foo@example.com"), Password: ptr.String("")},
			errContains: []string{"password"},
		},
		{
			payload:     UserPayload{Name: ptr.String("Foo"), Email: ptr.String("foo@example.com"), Password: ptr.String("foo")},
			errContains: []string{"password"},
		},
		{
			payload:     UserPayload{Name: ptr.String("Foo"), Email: ptr.String("foo@example.com"), Password: ptr.String("Foofoofoo1337#"), InviteToken: ptr.String("foo")},
			errContains: []string{"invite_token"},
		},
		{
			payload:     UserPayload{Name: ptr.String("Foo"), Email: ptr.String("foo@example.com"), Password: ptr.String("Foofoofoo1337#")},
			errContains: nil,
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			err := tc.payload.VerifyAdminCreate()
			if len(tc.errContains) == 0 {
				require.NoError(t, err)
			} else {
				ierr := err.(*InvalidArgumentError)
				require.Equal(t, len(tc.errContains), len(ierr.Errors))
				for _, expected := range tc.errContains {
					assertContainsErrorName(t, ierr, expected)
				}
			}
		})
	}
}

func TestInviteCreateValidate(t *testing.T) {
	testCases := []struct {
		payload UserPayload
		// if errContains is empty then no error is expected
		errContains []string
	}{
		{
			payload:     UserPayload{},
			errContains: []string{"name", "email", "password", "invite_token"},
		},
		{
			payload:     UserPayload{Name: ptr.String(""), Email: ptr.String(""), Password: ptr.String("")},
			errContains: []string{"name", "email", "password", "invite_token"},
		},
		{
			payload:     UserPayload{Name: ptr.String("Foo"), Email: ptr.String(""), Password: ptr.String("")},
			errContains: []string{"email", "password", "invite_token"},
		},
		{
			payload:     UserPayload{Name: ptr.String("Foo"), Email: ptr.String("foo@example.com"), Password: ptr.String("")},
			errContains: []string{"password", "invite_token"},
		},
		{
			payload:     UserPayload{Name: ptr.String("Foo"), Email: ptr.String("foo@example.com"), Password: ptr.String("foo")},
			errContains: []string{"password", "invite_token"},
		},
		{
			payload:     UserPayload{Name: ptr.String("Foo"), Email: ptr.String("foo@example.com"), Password: ptr.String("Foofoofoo1337#"), InviteToken: ptr.String("foo")},
			errContains: nil,
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			err := tc.payload.VerifyInviteCreate()
			if len(tc.errContains) == 0 {
				require.NoError(t, err)
			} else {
				ierr := err.(*InvalidArgumentError)
				for _, expected := range tc.errContains {
					require.Equal(t, len(tc.errContains), len(ierr.Errors))
					assertContainsErrorName(t, ierr, expected)
				}
			}
		})
	}
}

func TestValidateEmailError(t *testing.T) {
	errCases := []string{
		"invalid",
		"Name Surname <test@example.com>",
		"test.com",
		"",
	}

	for _, c := range errCases {
		require.Error(t, ValidateEmail(c))
	}
}

func TestValidateEmail(t *testing.T) {
	cases := []string{
		"user@example.com",
		"user@example.localhost",
		"user+1@example.com",
	}

	for _, c := range cases {
		require.NoError(t, ValidateEmail(c))
	}
}

func assertContainsErrorName(t *testing.T, invalid *InvalidArgumentError, name string) {
	for _, argErr := range invalid.Errors {
		if argErr.name == name {
			return
		}
	}
	t.Errorf("%v does not contain error %s", invalid, name)
}

func TestUserComputeStatus(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	daysAgo := func(days int) time.Time {
		return now.Add(-time.Duration(days) * 24 * time.Hour)
	}
	admin := new("admin")

	for _, tc := range []struct {
		name     string
		user     User
		expected UserStatus
	}{
		{
			name: "active when logged in recently",
			user: User{
				GlobalRole:  admin,
				LastLoginAt: new(daysAgo(1)),
			},
			expected: UserStatusActive,
		},
		{
			name: "inactive when last login is 30+ days old",
			user: User{
				GlobalRole:  admin,
				LastLoginAt: new(daysAgo(31)),
			},
			expected: UserStatusInactive,
		},
		{
			name: "inactive exactly at the 30-day boundary",
			user: User{
				GlobalRole:  admin,
				LastLoginAt: new(daysAgo(30)),
			},
			expected: UserStatusInactive,
		},
		{
			name: "falls back to created_at when never logged in, stale",
			user: User{
				UpdateCreateTimestamps: UpdateCreateTimestamps{
					CreateTimestamp: CreateTimestamp{CreatedAt: daysAgo(31)},
				},
				GlobalRole: admin,
			},
			expected: UserStatusInactive,
		},
		{
			name: "falls back to created_at when never logged in, fresh",
			user: User{
				UpdateCreateTimestamps: UpdateCreateTimestamps{
					CreateTimestamp: CreateTimestamp{CreatedAt: daysAgo(1)},
				},
				GlobalRole: admin,
			},
			expected: UserStatusActive,
		},
		{
			name: "active for API-only user with recent activity on an old token session",
			user: User{
				UpdateCreateTimestamps: UpdateCreateTimestamps{
					CreateTimestamp: CreateTimestamp{CreatedAt: daysAgo(100)},
				},
				GlobalRole:     admin,
				APIOnly:        true,
				LastLoginAt:    new(daysAgo(100)),
				LastActivityAt: new(daysAgo(1)),
			},
			expected: UserStatusActive,
		},
		{
			name: "inactive for API-only user with no recent activity",
			user: User{
				UpdateCreateTimestamps: UpdateCreateTimestamps{
					CreateTimestamp: CreateTimestamp{CreatedAt: daysAgo(100)},
				},
				GlobalRole:     admin,
				APIOnly:        true,
				LastLoginAt:    new(daysAgo(100)),
				LastActivityAt: new(daysAgo(45)),
			},
			expected: UserStatusInactive,
		},
		{
			name: "prefers most recent of last activity and last login",
			user: User{
				GlobalRole:     admin,
				LastLoginAt:    new(daysAgo(40)),
				LastActivityAt: new(daysAgo(2)),
			},
			expected: UserStatusActive,
		},
		{
			name: "no access when user has no roles, regardless of last login",
			user: User{
				LastLoginAt: new(daysAgo(1)),
			},
			expected: UserStatusNoAccess,
		},
		{
			name: "team role counts as access",
			user: User{
				Teams:       []UserTeam{{Team: Team{ID: 1}, Role: RoleObserver}},
				LastLoginAt: new(daysAgo(31)),
			},
			expected: UserStatusInactive,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.user.ComputeStatus(now))
		})
	}
}
