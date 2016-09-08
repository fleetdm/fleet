package server

import (
	"testing"

	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

func TestCreateUser(t *testing.T) {
	ds, _ := datastore.New("mock", "")
	svc, _ := NewService(testConfig(ds))
	ctx := context.Background()

	var createUserTests = []struct {
		Username           *string
		Password           *string
		Email              *string
		NeedsPasswordReset *bool
		Admin              *bool
		Err                error
	}{
		{
			Username: stringPtr("admin1"),
			Password: stringPtr("foobar"),
			Err:      invalidArgumentError{},
		},
		{
			Username:           stringPtr("admin1"),
			Password:           stringPtr("foobar"),
			Email:              stringPtr("admin1@example.com"),
			NeedsPasswordReset: boolPtr(true),
			Admin:              boolPtr(false),
		},
	}

	for _, tt := range createUserTests {
		payload := kolide.UserPayload{
			Username: tt.Username,
			Password: tt.Password,
			Email:    tt.Email,
			Admin:    tt.Admin,
			AdminForcedPasswordReset: tt.NeedsPasswordReset,
		}
		user, err := svc.NewUser(ctx, payload)
		switch err.(type) {
		case nil:
		case invalidArgumentError:
			continue
		default:
			t.Fatalf("got %q, want %q", err, tt.Err)
		}

		if user.ID == 0 {
			t.Errorf("expected a user ID, got 0")
		}

		if err := user.ValidatePassword(*tt.Password); err != nil {
			t.Errorf("expected nil, got %q", err)
		}

		if err := user.ValidatePassword("different_password!"); err == nil {
			t.Errorf("expected err, got nil")
		}

		if have, want := user.AdminForcedPasswordReset, *tt.NeedsPasswordReset; have != want {
			t.Errorf("have %v want %v", have, want)
		}

		if have, want := user.AdminForcedPasswordReset, *tt.NeedsPasswordReset; have != want {
			t.Errorf("have %v want %v", have, want)
		}

		if have, want := user.Admin, *tt.Admin; have != want {
			t.Errorf("have %v want %v", have, want)
		}

		// check duplicate creation
		_, err = svc.NewUser(ctx, payload)
		if err != datastore.ErrExists {
			t.Errorf("have %q, want %q", err, datastore.ErrExists)
		}
	}
}

func TestChangeUserPassword(t *testing.T) {
	ds, _ := datastore.New("mock", "")
	svc, _ := NewService(testConfig(ds))
	createTestUsers(t, ds)

	var passwordChangeTests = []struct {
		username        string
		currentPassword string
		newPassword     string
		err             error
	}{
		{
			username:        "admin1",
			currentPassword: *testUsers["admin1"].Password,
			newPassword:     "123cat!",
		},
	}

	ctx := context.Background()
	for _, tt := range passwordChangeTests {
		user, err := ds.User(tt.username)
		if err != nil {
			t.Fatal(err)
		}

		err = svc.ChangePassword(ctx, user.ID, tt.currentPassword, tt.newPassword)
		if err != nil {
			t.Fatal(err)
		}

	}
}

var testUsers = map[string]kolide.UserPayload{
	"admin1": {
		Username: stringPtr("admin1"),
		Password: stringPtr("foobar"),
		Email:    stringPtr("admin1@example.com"),
		Admin:    boolPtr(true),
	},
	"user1": {
		Username: stringPtr("user1"),
		Password: stringPtr("foobar"),
		Email:    stringPtr("user1@example.com"),
	},
	"user2": {
		Username: stringPtr("user2"),
		Password: stringPtr("bazfoo"),
		Email:    stringPtr("user2@example.com"),
	},
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
