package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jinzhu/gorm"
)

func TestNewUser(t *testing.T) {
	db, err := openTestDB()
	if err != nil {
		t.Fatal(err)
	}

	user, err := NewUser(db, "marpaia", "foobar", "mike@kolide.co", true, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	if user.Username != "marpaia" {
		t.Fatalf("Username is not what's expected: %s", user.Username)
	}

	if user.Email != "mike@kolide.co" {
		t.Fatalf("Email is not what's expected: %s", user.Email)
	}

	if !user.Admin {
		t.Fatal("User is not an admin")
	}

	var verify User
	db.Where("username = ?", "marpaia").First(&verify)
	if verify.ID != user.ID {
		t.Fatal("Couldn't select user back from database")
	}
}

func TestValidatePassword(t *testing.T) {
	db, err := openTestDB()
	if err != nil {
		t.Fatal(err.Error())
	}

	user, err := NewUser(db, "marpaia", "foobar", "mike@kolide.co", true, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = user.ValidatePassword("foobar")
	if err != nil {
		t.Fatal("Password validation failed")
	}

	err = user.ValidatePassword("not correct")
	if err == nil {
		t.Fatal("Incorrect password worked")
	}
}

func TestMakeAdmin(t *testing.T) {
	db, err := openTestDB()
	if err != nil {
		t.Fatal(err.Error())
	}

	user, err := NewUser(db, "marpaia", "foobar", "mike@kolide.co", false, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	if user.Admin {
		t.Fatal("Admin should be false")
	}

	err = user.MakeAdmin(db)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !user.Admin {
		t.Fatal("Admin should be true")
	}

	var verify User
	db.Where("admin = ?", true).First(&verify)

	if user.ID != verify.ID {
		t.Fatal("Users don't match")
	}

	if !verify.Admin {
		t.Fatal("User wasn't set as admin in the database")
	}

}

func TestUpdatingUser(t *testing.T) {
	db, err := openTestDB()
	if err != nil {
		t.Fatal(err.Error())
	}

	user, err := NewUser(db, "marpaia", "foobar", "mike@kolide.co", false, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	user.Email = "marpaia@kolide.co"
	err = db.Save(user).Error
	if err != nil {
		t.Fatal(err.Error())
	}

	if user.Email != "marpaia@kolide.co" {
		t.Fatal("user.Email was reset")
	}

	var verify User
	err = db.Where("id = ?", user.ID).First(&verify).Error
	if err != nil {
		t.Fatal(err.Error())
	}

	if verify.Email != "marpaia@kolide.co" {
		t.Fatalf("user's email was not updated in the DB: %s", verify.Email)
	}

}

func TestDeletingUser(t *testing.T) {
	db, err := openTestDB()
	if err != nil {
		t.Fatal(err.Error())
	}

	user, err := NewUser(db, "marpaia", "foobar", "mike@kolide.co", false, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	var verify1 User
	err = db.Where("username = ?", "marpaia").First(&verify1).Error
	if err != nil {
		t.Fatal(err.Error())
	}
	if verify1.ID != user.ID {
		t.Fatal("users are not the same")
	}

	err = db.Delete(&user).Error
	if err != nil {
		t.Fatal(err.Error())
	}

	var verify2 User
	err = db.Where("username = ?", "marpaia").First(&verify2).Error
	if err != gorm.ErrRecordNotFound {
		t.Fatal("Record was not deleted")
	}
}

func TestSetPassword(t *testing.T) {
	db, err := openTestDB()
	if err != nil {
		t.Fatal(err.Error())
	}

	user, err := NewUser(db, "marpaia", "foobar", "mike@kolide.co", false, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = user.ValidatePassword("foobar")
	if err != nil {
		t.Fatal(err.Error())
	}

	err = user.SetPassword(db, "baz")
	if err != nil {
		t.Fatal(err.Error())
	}

	err = user.ValidatePassword("baz")
	if err != nil {
		t.Fatal(err.Error())
	}

	var verify User
	err = db.Where("username = ?", "marpaia").First(&verify).Error
	if err != nil {
		t.Fatal(err.Error())
	}

	err = verify.ValidatePassword("baz")
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestUserManagementIntegration(t *testing.T) {
	r := createTestServer()
	r.Use(testSessionMiddleware)
	r.Use(JWTRenewalMiddleware)

	db, err := openTestDB()
	if err != nil {
		t.Fatal(err.Error())
	}
	injectedTestDB = db

	admin, err := NewUser(db, "admin", "foobar", "admin@kolide.co", true, false)
	if err != nil {
		t.Fatal(err.Error())
	}
	_ = admin

	r.POST("/login", Login)
	r.GET("/logout", Logout)

	r.GET("/user", GetUser)
	r.PUT("/user", CreateUser)
	r.PATCH("/user", ModifyUser)
	r.DELETE("/user", DeleteUser)

	res1 := httptest.NewRecorder()
	body1, err := json.Marshal(LoginRequestBody{
		Username: "admin",
		Password: "foobar",
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	buff1 := new(bytes.Buffer)
	buff1.Write(body1)
	req1, _ := http.NewRequest("POST", "/login", buff1)
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(res1, req1)
	if res1.Code != 200 {
		t.Fatalf("Response code: %d", res1.Code)
	}

	res2 := httptest.NewRecorder()
	body2, err := json.Marshal(CreateUserRequestBody{
		Username:           "marpaia",
		Password:           "foobar",
		Email:              "mike@kolide.co",
		Admin:              false,
		NeedsPasswordReset: false,
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	buff2 := new(bytes.Buffer)
	buff2.Write(body2)
	req2, _ := http.NewRequest("PUT", "/user", buff2)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Cookie", res1.Header().Get("Set-Cookie"))
	r.ServeHTTP(res2, req2)

	res3 := httptest.NewRecorder()
	body3, err := json.Marshal(CreateUserRequestBody{
		Username:           "admin2",
		Password:           "foobar",
		Email:              "admin2@kolide.co",
		Admin:              true,
		NeedsPasswordReset: false,
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	buff3 := new(bytes.Buffer)
	buff3.Write(body3)
	req3, _ := http.NewRequest("PUT", "/user", buff3)
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Cookie", res1.Header().Get("Set-Cookie"))
	r.ServeHTTP(res3, req3)

	var user User
	err = db.Where("username = ?", "marpaia").First(&user).Error
	if err != nil {
		t.Fatal(err.Error())
	}

	if user.Email != "mike@kolide.co" {
		t.Fatalf("user's email was not set in the DB: %s", user.Email)
	}
	if user.Admin {
		t.Fatal("user shouldn't be admin")
	}

	var admin2 User
	err = db.Where("username = ?", "admin2").First(&admin2).Error
	if err != nil {
		t.Fatal(err.Error())
	}

	if admin2.Email != "admin2@kolide.co" {
		t.Fatalf("admin2's email was not set in the DB: %s", admin2.Email)
	}
	if !admin2.Admin {
		t.Fatal("admin2 should be admin")
	}

}
