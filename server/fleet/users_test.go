package fleet

import (
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

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
	passwordTests := []string{"foobar!!", "bazbing!!"}
	keySize := 24
	cost := 10

	for _, pwd := range passwordTests {
		hashed, salt, err := saltAndHashPassword(keySize, pwd, cost)
		require.NoError(t, err)

		saltAndPass := []byte(fmt.Sprintf("%s%s", pwd, salt))
		err = bcrypt.CompareHashAndPassword(hashed, saltAndPass)
		require.NoError(t, err)

		err = bcrypt.CompareHashAndPassword(hashed, []byte(fmt.Sprint("invalidpassword", salt)))
		require.Error(t, err)
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
				require.Equal(t, len(tc.errContains), len(*ierr))
				for _, expected := range tc.errContains {
					assertContainsErrorName(t, *ierr, expected)
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
					require.Equal(t, len(tc.errContains), len(*ierr))
					assertContainsErrorName(t, *ierr, expected)
				}
			}
		})
	}
}

func assertContainsErrorName(t *testing.T, invalid InvalidArgumentError, name string) {
	for _, argErr := range invalid {
		if argErr.name == name {
			return
		}
	}
	t.Errorf("%v does not contain error %s", invalid, name)
}
