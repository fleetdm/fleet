package main

import (
	"testing"

	"github.com/jinzhu/gorm"
)

func TestNewUser(t *testing.T) {
	db := openTestDB(t)

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
	db := openTestDB(t)

	user, err := NewUser(db, "marpaia", "foobar", "mike@kolide.co", true, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	{
		err := user.ValidatePassword("foobar")
		if err != nil {
			t.Error("Password validation failed")
		}
	}

	{
		err := user.ValidatePassword("different")
		if err == nil {
			t.Error("Incorrect password worked")
		}
	}
}

func TestMakeAdmin(t *testing.T) {
	db := openTestDB(t)

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
	db := openTestDB(t)

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
	db := openTestDB(t)

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
	db := openTestDB(t)

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
