package kolide

import "testing"

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
		if err != nil {
			t.Fatalf("error creating new user: %s", err)
		}

		{
			err := user.ValidatePassword(tt.Password)
			if err != nil {
				t.Errorf("Password validation failed for user %s", user.Username)
			}
		}

		{
			err := user.ValidatePassword("different")
			if err == nil {
				t.Errorf("Incorrect password worked for user %s", user.Username)
			}
		}
	}
}
