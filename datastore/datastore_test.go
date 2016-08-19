package datastore

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/kolide"
)

func TestPasswordResetRequests(t *testing.T) {
	db := setup(t)
	defer teardown(t, db)

	testPasswordResetRequests(t, db)
}

func testPasswordResetRequests(t *testing.T, db Datastore) {
	createTestUsers(t, db)
	now := time.Now()
	tomorrow := now.Add(time.Hour * 24)
	var passwordResetTests = []struct {
		userID  uint
		expires time.Time
		token   string
	}{
		{userID: 1, expires: tomorrow, token: "abcd"},
	}

	for _, tt := range passwordResetTests {
		req, err := db.CreatePassworResetRequest(tt.userID, tt.expires, tt.token)
		if err != nil {
			t.Fatalf("failed to create PasswordResetRequest campaign in datastore")
		}

		if req.UserID != tt.userID {
			t.Fatalf("expected %v, got %v", tt.userID, req.UserID)
		}

	}
}

// TestUser tests the UserStore interface
// this test uses the default testing backend
func TestCreateUser(t *testing.T) {
	db := setup(t)
	defer teardown(t, db)

	testCreateUser(t, db)
}

func TestSaveUser(t *testing.T) {
	db := setup(t)
	defer teardown(t, db)

	testSaveUser(t, db)
}

func testCreateUser(t *testing.T, db kolide.UserStore) {
	var createTests = []struct {
		username, password, email string
		isAdmin, passwordReset    bool
	}{
		{"marpaia", "foobar", "mike@kolide.co", true, false},
		{"jason", "foobar", "jason@kolide.co", true, false},
	}

	for _, tt := range createTests {
		u, err := kolide.NewUser(tt.username, tt.password, tt.email, tt.isAdmin, tt.passwordReset)
		if err != nil {
			t.Fatal(err)
		}

		user, err := db.NewUser(u)
		if err != nil {
			t.Fatal(err)
		}

		verify, err := db.User(tt.username)
		if err != nil {
			t.Fatal(err)
		}

		if verify.ID != user.ID {
			t.Fatalf("expected %q, got %q", user.ID, verify.ID)
		}

		if verify.Username != tt.username {
			t.Errorf("expected username: %s, got %s", tt.username, verify.Username)
		}

		if verify.Email != tt.email {
			t.Errorf("expected email: %s, got %s", tt.email, verify.Email)
		}

		if verify.Admin != tt.isAdmin {
			t.Errorf("expected email: %s, got %s", tt.email, verify.Email)
		}
	}
}

func TestUserByID(t *testing.T) {
	db := setup(t)
	defer teardown(t, db)

	testUserByID(t, db)
}

func testUserByID(t *testing.T, db kolide.UserStore) {
	users := createTestUsers(t, db)
	for _, tt := range users {
		returned, err := db.UserByID(tt.ID)
		if err != nil {
			t.Fatal(err)
		}

		if returned.ID != tt.ID {
			t.Errorf("expected ID %v, got %v", tt.ID, returned.ID)
		}
	}

	// test missing user
	_, err := db.UserByID(10000000000)
	if err == nil {
		t.Errorf("expected error for missing user, got nil")
	}

}

func createTestUsers(t *testing.T, db kolide.UserStore) []*kolide.User {
	var createTests = []struct {
		username, password, email string
		isAdmin, passwordReset    bool
	}{
		{"marpaia", "foobar", "mike@kolide.co", true, false},
		{"jason", "foobar", "jason@kolide.co", false, false},
	}

	var users []*kolide.User
	for _, tt := range createTests {
		u, err := kolide.NewUser(tt.username, tt.password, tt.email, tt.isAdmin, tt.passwordReset)
		if err != nil {
			t.Fatal(err)
		}

		user, err := db.NewUser(u)
		if err != nil {
			t.Fatal(err)
		}

		users = append(users, user)
	}
	if len(users) == 0 {
		t.Fatal("expected a list of users, got 0")
	}
	return users
}

func testSaveUser(t *testing.T, db kolide.UserStore) {
	users := createTestUsers(t, db)
	testAdminAttribute(t, db, users)
	testEmailAttribute(t, db, users)
	testPasswordAttribute(t, db, users)
}

func testPasswordAttribute(t *testing.T, db kolide.UserStore, users []*kolide.User) {
	for _, user := range users {
		user.Password = []byte(randomString(8))
		err := db.SaveUser(user)
		if err != nil {
			t.Fatalf("failed to save user %s", user.Name)
		}

		verify, err := db.User(user.Username)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(verify.Password, user.Password) {
			t.Errorf("expected password attribute to be %v, got %v", user.Password, verify.Password)
		}

	}
}

func testEmailAttribute(t *testing.T, db kolide.UserStore, users []*kolide.User) {
	for _, user := range users {
		user.Email = fmt.Sprintf("test.%s", user.Email)
		err := db.SaveUser(user)
		if err != nil {
			t.Fatalf("failed to save user %s", user.Name)
		}

		verify, err := db.User(user.Username)
		if err != nil {
			t.Fatal(err)
		}

		if verify.Email != user.Email {
			t.Errorf("expected admin attribute to be %v, got %v", user.Email, verify.Email)
		}
	}
}

func testAdminAttribute(t *testing.T, db kolide.UserStore, users []*kolide.User) {
	for _, user := range users {
		user.Admin = false
		err := db.SaveUser(user)
		if err != nil {
			t.Fatalf("failed to save user %s", user.Name)
		}

		verify, err := db.User(user.Username)
		if err != nil {
			t.Fatal(err)
		}

		if verify.Admin != user.Admin {
			t.Errorf("expected admin attribute to be %v, got %v", user.Admin, verify.Admin)
		}
	}
}

// setup creates a datastore for testing
func setup(t *testing.T) Datastore {
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("error opening test db: %s", err)
	}
	ds := gormDB{DB: db}
	if err := ds.Migrate(); err != nil {
		t.Fatal(err)
	}
	// Log using t.Log so that output only shows up if the test fails
	// db.SetLogger(&testLogger{t: t})
	// db.LogMode(true)
	return ds
}

func teardown(t *testing.T, ds Datastore) {
	if err := ds.Drop(); err != nil {
		t.Fatal(err)
	}
}

type testLogger struct {
	t *testing.T
}

func (t *testLogger) Print(v ...interface{}) {
	t.t.Log(v...)
}

func (t *testLogger) Write(p []byte) (n int, err error) {
	t.t.Log(string(p))
	return len(p), nil
}

func randomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
