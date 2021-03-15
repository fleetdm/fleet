package kolide

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestValidatePassword(t *testing.T) {

	var passwordTests = []struct {
		Username, Password, Email string
		Admin, PasswordReset      bool
	}{
		{"marpaia", "foobar", "mike@kolide.co", true, false},
		{"jason", "bar0baz!?", "jason@kolide.co", true, false},
	}

	for _, tt := range passwordTests {
		user := newTestUser(t, tt.Username, tt.Password, tt.Email)

		err := user.ValidatePassword(tt.Password)
		assert.Nil(t, err)

		err = user.ValidatePassword("different")
		assert.NotNil(t, err)
	}
}

func newTestUser(t *testing.T, username, password, email string) *User {
	var (
		salt = "test-salt"
		cost = 10
	)
	withSalt := []byte(fmt.Sprintf("%s%s", password, salt))
	hashed, _ := bcrypt.GenerateFromPassword(withSalt, cost)
	return &User{
		Username: username,
		Salt:     salt,
		Password: hashed,
		Email:    email,
	}
}
