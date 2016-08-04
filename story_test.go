package main

import (
	"strings"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
)

func TestUserAndAccountManagement(t *testing.T) {

	// Create and configure the webserver which will be used to handle the tests
	var req IntegrationRequests
	req.New(t)

	// Instantiate the variables that will store the most recent session cookie
	// for each user context that will be created
	var adminSession string
	var admin2Session string
	var user1Session string
	var user2Session string

	// Test logging in with the first admin
	req.Login("admin", "foobar", &adminSession)

	// Once admin is logged in, create a user using a valid admin session
	req.CreateAndCheckUser("user1", "foobar", "user1@kolide.co", "", false, false, adminSession)

	// Once admin is logged in, create another admin account using a valid
	// admin session
	req.CreateAndCheckUser("admin2", "foobar", "admin2@kolide.co", "", true, false, adminSession)

	// Once admin has created admin2, log in with admin2 to get a session
	// context for admin2
	req.Login("admin2", "foobar", &admin2Session)

	// Use an admin created via the API to create a user via the API
	req.CreateAndCheckUser("user2", "foobar", "user2@kolide.co", "", false, false, admin2Session)

	// Once admin has created user1, log in with user1 to get a session context
	// for user1
	req.Login("user1", "foobar", &user1Session)

	// Once admin2 has created user2, log in with user1 to get a session context
	// for user2
	req.Login("user2", "foobar", &user2Session)

	// Get info on user2 as admin2
	req.GetAndCheckUser("user2", admin2Session)

	// Get info on admin2 as user2
	req.GetAndCheckUser("admin2", user2Session)

	// Get session info for admin
	adminSessionInfo := req.GetUserSessionInfo("admin", adminSession)
	if len(adminSessionInfo.Sessions) != 1 {
		t.Fatalf("Expected 1 session, found %d", len(adminSessionInfo.Sessions))
	}

	// Pull the token out of the JWT token and get the session info via that
	token, err := ParseJWT(strings.Split(adminSession, "=")[1])
	if err != nil {
		t.Fatal(err.Error())
	}
	sessionKey := token.Claims.(jwt.MapClaims)["session_key"].(string)

	adminSessionInfoVerify := req.GetSessionInfo(sessionKey, adminSession)

	if adminSessionInfo.Sessions[0].SessionID != adminSessionInfoVerify.SessionID {
		t.Fatal("Session IDs don't match")
	}

	// Delete the admin session
	req.DeleteSession(adminSessionInfo.Sessions[0].SessionID, adminSession)

	// Verify the session was deleted
	sessionVerify := &Session{
		Key: sessionKey,
	}
	err = req.db.Where(sessionVerify).First(sessionVerify).Error
	if err != gorm.ErrRecordNotFound {
		t.Fatal("Record should not exist in the database")
	}

	// Re-login as admin
	req.Login("admin", "foobar", &adminSession)
	var adminSession2 string
	req.Login("admin", "foobar", &adminSession2)

	// Get session info for admin
	adminSessionInfo = req.GetUserSessionInfo("admin", adminSession)
	if len(adminSessionInfo.Sessions) != 2 {
		t.Fatalf("Expected 2 sessions, found %d", len(adminSessionInfo.Sessions))
	}

	// Delete all admin session as admin2
	req.DeleteUserSessions("admin", admin2Session)

	// Verify there are no admin sessions left
	adminSessionInfo = req.GetUserSessionInfo("admin", admin2Session)
	if len(adminSessionInfo.Sessions) != 0 {
		t.Fatalf("Expected 0 sessions, found %d", len(adminSessionInfo.Sessions))
	}

	// Re-login as admin
	req.Login("admin", "foobar", &adminSession)

	// Modify user1 as admin
	req.ModifyAndCheckUser("user1", "user1@kolide.co", "User One", false, false, adminSession)

	// Modify user2 as user2
	req.ModifyAndCheckUser("user2", "user2@kolide.co", "User Two", false, false, user2Session)

	// admin resets user1 password
	req.ChangePassword("user1", "", "bazz1", adminSession)

	// user1 logs in with new password
	req.Login("user1", "bazz1", &user1Session)

	// user2 resets user2 password
	req.ChangePassword("user2", "foobar", "bazz2", user2Session)

	// user2 logs in with new password
	req.Login("user2", "bazz2", &user2Session)

	// admin2 promotes user2 to admin
	req.SetAdminStateAndCheckUser("user2", true, admin2Session)

	// user2 is admin
	resp := req.GetUser("user2", user2Session)
	if !resp.Admin {
		t.Fatal("user2 should be an admin")
	}

	// admin demotes user2 from admin
	req.SetAdminStateAndCheckUser("user2", false, adminSession)

	// user2 is no longer an admin
	resp = req.GetUser("user2", user2Session)
	if resp.Admin {
		t.Fatal("user2 shouldn't be an admin")
	}

	// admin sets user1 as no longer enabled
	req.SetEnabledStateAndCheckUser("user1", false, adminSession)

	// user1 is no longer enabled
	resp = req.GetUser("user1", user2Session)
	if resp.Enabled {
		t.Fatal("user1 shouldn't be enabled")
	}

	// admin2 re-enables user1
	req.SetEnabledStateAndCheckUser("user1", true, admin2Session)

	// user1 can view user2
	req.GetUser("user2", user2Session)

	// Delete admin2 as admin1
	req.DeleteAndCheckUser("admin2", adminSession)

	// Delete user2 as admin
	req.DeleteAndCheckUser("user2", adminSession)
}
