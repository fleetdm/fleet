package kolide

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePassword(t *testing.T) {

	var passwordTests = []struct {
		Username, Password, Email string
		Admin, PasswordReset      bool
	}{
		{"marpaia", "foobar", "mike@kolide.co", true, false},
		{"jason", "bar0baz!?", "jason@kolide.co", true, false},
	}

	const bcryptCost = 6

	for _, tt := range passwordTests {
		user, err := NewUser(tt.Username, tt.Password, tt.Email, tt.Admin, tt.PasswordReset, bcryptCost)
		assert.Nil(t, err)

		err = user.ValidatePassword(tt.Password)
		assert.Nil(t, err)

		err = user.ValidatePassword("different")
		assert.NotNil(t, err)
	}
}
