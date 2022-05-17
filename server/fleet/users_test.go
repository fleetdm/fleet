package fleet

import (
	"fmt"
	"testing"

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
	}
}
