package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type integrationRequests struct {
	r  *gin.Engine
	db *gorm.DB
	t  *testing.T
}

func (req *integrationRequests) New(t *testing.T) {
	req.t = t

	req.r = createTestServer()
	req.r.Use(testSessionMiddleware)
	req.r.Use(JWTRenewalMiddleware)

	req.db, _ = openTestDB()
	injectedTestDB = req.db

	// Until we have a better solution for first-user onboarding, manually
	// create an admin
	_, err := NewUser(req.db, "admin", "foobar", "admin@kolide.co", true, false)
	if err != nil {
		panic(err.Error())
	}

	req.r.POST("/login", Login)
	req.r.GET("/logout", Logout)

	req.r.POST("/user", GetUser)
	req.r.PUT("/user", CreateUser)
	req.r.PATCH("/user", ModifyUser)
	req.r.DELETE("/user", DeleteUser)

	req.r.PATCH("/user/password", ChangeUserPassword)
	req.r.PATCH("/user/admin", SetUserAdminState)
	req.r.PATCH("/user/enabled", SetUserEnabledState)
}

func (req *integrationRequests) Login(username, password string, sessionOut *string) {
	response := httptest.NewRecorder()
	body, err := json.Marshal(LoginRequestBody{
		Username: username,
		Password: password,
	})
	if err != nil {
		req.t.Fatal(err.Error())
		return
	}

	buff := new(bytes.Buffer)
	buff.Write(body)
	request, _ := http.NewRequest("POST", "/login", buff)
	request.Header.Set("Content-Type", "application/json")
	req.r.ServeHTTP(response, request)

	if response.Code != 200 {
		req.t.Fatalf("Response code: %d", response.Code)
		return
	}
	*sessionOut = response.Header().Get("Set-Cookie")

	return
}

func (req *integrationRequests) CreateUser(username, password, email string, admin, reset bool, session *string) *GetUserResponseBody {
	response := httptest.NewRecorder()
	body, err := json.Marshal(CreateUserRequestBody{
		Username:           username,
		Password:           password,
		Email:              email,
		Admin:              admin,
		NeedsPasswordReset: reset,
	})
	if err != nil {
		req.t.Fatal(err.Error())
		return nil
	}

	buff := new(bytes.Buffer)
	buff.Write(body)
	request, _ := http.NewRequest("PUT", "/user", buff)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", *session)
	req.r.ServeHTTP(response, request)

	if response.Code != 200 {
		req.t.Fatalf("Response code: %d", response.Code)
		return nil
	}
	*session = response.Header().Get("Set-Cookie")

	var responseBody GetUserResponseBody
	err = json.Unmarshal(response.Body.Bytes(), &responseBody)
	if err != nil {
		req.t.Fatal(err.Error())
		return nil
	}

	return &responseBody
}

func (req *integrationRequests) GetUser(username string, session *string) *GetUserResponseBody {
	response := httptest.NewRecorder()
	body, err := json.Marshal(GetUserRequestBody{
		Username: username,
	})
	if err != nil {
		req.t.Fatal(err.Error())
		return nil
	}

	buff := new(bytes.Buffer)
	buff.Write(body)
	request, _ := http.NewRequest("POST", "/user", buff)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", *session)
	req.r.ServeHTTP(response, request)

	if response.Code != 200 {
		req.t.Fatalf("Response code: %d", response.Code)
		return nil
	}
	*session = response.Header().Get("Set-Cookie")

	var responseBody GetUserResponseBody
	err = json.Unmarshal(response.Body.Bytes(), &responseBody)
	if err != nil {
		req.t.Fatal(err.Error())
		return nil
	}

	return &responseBody
}

func (req *integrationRequests) ModifyUser(username, name, email string, session *string) *GetUserResponseBody {
	response := httptest.NewRecorder()
	body, err := json.Marshal(ModifyUserRequestBody{
		Username: username,
		Name:     name,
		Email:    email,
	})
	if err != nil {
		req.t.Fatal(err.Error())
		return nil
	}

	buff := new(bytes.Buffer)
	buff.Write(body)
	request, _ := http.NewRequest("PATCH", "/user", buff)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", *session)
	req.r.ServeHTTP(response, request)

	if response.Code != 200 {
		req.t.Fatalf("Response code: %d", response.Code)
		return nil
	}
	*session = response.Header().Get("Set-Cookie")

	var responseBody GetUserResponseBody
	err = json.Unmarshal(response.Body.Bytes(), &responseBody)
	if err != nil {
		req.t.Fatal(err.Error())
		return nil
	}

	return &responseBody
}

func (req *integrationRequests) DeleteUser(username string, session *string) {
	response := httptest.NewRecorder()
	body, err := json.Marshal(DeleteUserRequestBody{
		Username: username,
	})
	if err != nil {
		req.t.Fatal(err.Error())
		return
	}

	buff := new(bytes.Buffer)
	buff.Write(body)
	request, _ := http.NewRequest("DELETE", "/user", buff)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", *session)
	req.r.ServeHTTP(response, request)

	if response.Code != 200 {
		req.t.Fatalf("Response code: %d", response.Code)
		return
	}
	*session = response.Header().Get("Set-Cookie")

	return
}

func (req *integrationRequests) ChangePassword(username, currentPassword, newPassword string, session *string) *GetUserResponseBody {
	response := httptest.NewRecorder()
	body, err := json.Marshal(ChangePasswordRequestBody{
		Username:          username,
		CurrentPassword:   currentPassword,
		NewPassword:       newPassword,
		NewPasswordConfim: newPassword,
	})
	if err != nil {
		req.t.Fatal(err.Error())
		return nil
	}

	buff := new(bytes.Buffer)
	buff.Write(body)
	request, _ := http.NewRequest("PATCH", "/user/password", buff)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", *session)
	req.r.ServeHTTP(response, request)

	if response.Code != 200 {
		req.t.Fatalf("Response code: %d", response.Code)
		return nil
	}
	*session = response.Header().Get("Set-Cookie")

	var responseBody GetUserResponseBody
	err = json.Unmarshal(response.Body.Bytes(), &responseBody)
	if err != nil {
		req.t.Fatal(err.Error())
	}

	return &responseBody
}

func (req *integrationRequests) SetAdminState(username string, admin bool, session *string) *GetUserResponseBody {
	response := httptest.NewRecorder()
	body, err := json.Marshal(SetUserAdminStateRequestBody{
		Username: username,
		Admin:    admin,
	})
	if err != nil {
		req.t.Fatal(err.Error())
		return nil
	}

	buff := new(bytes.Buffer)
	buff.Write(body)
	request, _ := http.NewRequest("PATCH", "/user/admin", buff)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", *session)
	req.r.ServeHTTP(response, request)

	if response.Code != 200 {
		req.t.Fatalf("Response code: %d", response.Code)
		return nil
	}
	*session = response.Header().Get("Set-Cookie")

	var responseBody GetUserResponseBody
	err = json.Unmarshal(response.Body.Bytes(), &responseBody)
	if err != nil {
		req.t.Fatal(err.Error())
	}

	return &responseBody
}

func (req *integrationRequests) SetEnabledState(username string, enabled bool, session *string) *GetUserResponseBody {
	response := httptest.NewRecorder()
	body, err := json.Marshal(SetUserEnabledStateRequestBody{
		Username: username,
		Enabled:  enabled,
	})
	if err != nil {
		req.t.Fatal(err.Error())
		return nil
	}

	buff := new(bytes.Buffer)
	buff.Write(body)
	request, _ := http.NewRequest("PATCH", "/user/enabled", buff)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", *session)
	req.r.ServeHTTP(response, request)

	if response.Code != 200 {
		req.t.Fatalf("Response code: %d", response.Code)
		return nil
	}
	*session = response.Header().Get("Set-Cookie")

	var responseBody GetUserResponseBody
	err = json.Unmarshal(response.Body.Bytes(), &responseBody)
	if err != nil {
		req.t.Fatal(err.Error())
		return nil
	}

	return &responseBody
}

func (req *integrationRequests) CheckUser(username, email, name string, admin, reset, enabled bool) {
	var user User
	err := req.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		req.t.Fatal(err.Error())
		return
	}
	if user.Email != email {
		req.t.Fatalf("user's email was not set in the DB: %s", user.Email)
	}
	if user.Admin != admin {
		req.t.Fatal("user admin settings don't match")
	}
	if user.NeedsPasswordReset != reset {
		req.t.Fatal("user reset settings don't match")
	}
	if user.Enabled != enabled {
		req.t.Fatal("user enabled settings don't match")
	}
	if user.Name != name {
		req.t.Fatalf("user names don't match: %s and %s", user.Name, name)
	}
	return
}

func (req *integrationRequests) GetAndCheckUser(username string, session *string) {
	resp := req.GetUser(username, session)
	req.CheckUser(username, resp.Email, resp.Name, resp.Admin, resp.NeedsPasswordReset, resp.Enabled)
}

func (req *integrationRequests) CreateAndCheckUser(username, password, email, name string, admin, reset bool, session *string) {
	resp := req.CreateUser(username, password, email, admin, reset, session)
	req.CheckUser(username, email, name, admin, reset, resp.Enabled)
}

func (req *integrationRequests) ModifyAndCheckUser(username, email, name string, admin, reset bool, session *string) {
	resp := req.ModifyUser(username, name, email, session)
	req.CheckUser(username, email, name, admin, reset, resp.Enabled)
}

func (req *integrationRequests) DeleteAndCheckUser(username string, session *string) {
	req.DeleteUser(username, session)

	var user User
	err := req.db.Where("username = ?", username).First(&user).Error
	if err == nil {
		req.t.Fatal("User should have been deleted.")
	}
}

func (req *integrationRequests) SetEnabledStateAndCheckUser(username string, enabled bool, session *string) {
	resp := req.SetEnabledState(username, enabled, session)
	req.CheckUser(username, resp.Email, resp.Name, resp.Admin, resp.NeedsPasswordReset, enabled)
}

func (req *integrationRequests) SetAdminStateAndCheckUser(username string, admin bool, session *string) {
	resp := req.SetAdminState(username, admin, session)
	req.CheckUser(username, resp.Email, resp.Name, admin, resp.NeedsPasswordReset, resp.Enabled)
}

func TestUserAndAccountManagement(t *testing.T) {

	// Create and configure the webserver which will be used to handle the tests
	var req integrationRequests
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
	req.CreateAndCheckUser("user1", "foobar", "user1@kolide.co", "", false, false, &adminSession)

	// Once admin is logged in, create another admin account using a valid
	// admin session
	req.CreateAndCheckUser("admin2", "foobar", "admin2@kolide.co", "", true, false, &adminSession)

	// Once admin has created admin2, log in with admin2 to get a session
	// context for admin2
	req.Login("admin2", "foobar", &admin2Session)

	// Use an admin created via the API to create a user via the API
	req.CreateAndCheckUser("user2", "foobar", "user2@kolide.co", "", false, false, &admin2Session)

	// Once admin has created user1, log in with user1 to get a session context
	// for user1
	req.Login("user1", "foobar", &user1Session)

	// Once admin2 has created user2, log in with user1 to get a session context
	// for user2
	req.Login("user2", "foobar", &user2Session)

	// Get info on user2 as admin2
	req.GetAndCheckUser("user2", &admin2Session)

	// Get info on admin2 as user2
	req.GetAndCheckUser("admin2", &user2Session)

	// Modify user1 as admin
	req.ModifyAndCheckUser("user1", "user1@kolide.co", "User One", false, false, &adminSession)

	// Modify user2 as user2
	req.ModifyAndCheckUser("user2", "user2@kolide.co", "User Two", false, false, &user2Session)

	// admin resets user1 password
	req.ChangePassword("user1", "", "bazz1", &adminSession)

	// user1 logs in with new password
	req.Login("user1", "bazz1", &user1Session)

	// user2 resets user2 password
	req.ChangePassword("user2", "foobar", "bazz2", &user2Session)

	// user2 logs in with new password
	req.Login("user2", "bazz2", &user2Session)

	// admin2 promotes user2 to admin
	req.SetAdminStateAndCheckUser("user2", true, &admin2Session)

	// user2 is admin
	resp := req.GetUser("user2", &user2Session)
	if !resp.Admin {
		t.Fatal("user2 should be an admin")
	}

	// admin demotes user2 from admin
	req.SetAdminStateAndCheckUser("user2", false, &adminSession)

	// user2 is no longer an admin
	resp = req.GetUser("user2", &user2Session)
	if resp.Admin {
		t.Fatal("user2 shouldn't be an admin")
	}

	// admin sets user1 as no longer enabled
	req.SetEnabledStateAndCheckUser("user1", false, &adminSession)

	// user1 is no longer enabled
	resp = req.GetUser("user1", &user2Session)
	if resp.Enabled {
		t.Fatal("user1 shouldn't be enabled")
	}

	// admin2 re-enables user1
	req.SetEnabledStateAndCheckUser("user1", true, &admin2Session)

	// user1 can view user2
	req.GetUser("user2", &user2Session)

	// Delete admin2 as admin1
	req.DeleteAndCheckUser("admin2", &adminSession)

	// Delete user2 as admin
	req.DeleteAndCheckUser("user2", &adminSession)
}
