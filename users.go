package main

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

// User is the model struct which represents a kolide user
type User struct {
	BaseModel
	Username           string `gorm:"not null;unique_index:idx_user_unique_username"`
	Password           []byte `gorm:"not null"`
	Salt               string `gorm:"not null"`
	Name               string
	Email              string `gorm:"not null;unique_index:idx_user_unique_email"`
	Admin              bool   `gorm:"not null"`
	Enabled            bool   `gorm:"not null"`
	NeedsPasswordReset bool
}

// NewUser is a wrapper around the creation of a new user.
// NewUser exists largely to allow the API to simply accept a string password
// while using the applications password hashing mechanisms to salt and hash the
// password.
func NewUser(db *gorm.DB, username, password, email string, admin, needsPasswordReset bool) (*User, error) {
	salt, hash, err := SaltAndHashPassword(password)
	if err != nil {
		return nil, err
	}
	user := &User{
		Username:           username,
		Password:           hash,
		Salt:               salt,
		Email:              email,
		Admin:              admin,
		Enabled:            true,
		NeedsPasswordReset: needsPasswordReset,
	}

	err = db.Create(&user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

// ValidatePassword accepts a potential password for a given user and attempts
// to validate it against the hash stored in the database after joining the
// supplied password with the stored password salt
func (u *User) ValidatePassword(password string) error {
	saltAndPass := []byte(fmt.Sprintf("%s%s", u.Salt, password))
	return bcrypt.CompareHashAndPassword(u.Password, saltAndPass)
}

// SetPassword accepts a new password for a user object and updates the salt
// and hash for that user in the database. This function assumes that the
// appropriate authorization checks have already occurred by the caller.
func (u *User) SetPassword(db *gorm.DB, password string) error {
	salt, hash, err := SaltAndHashPassword(password)
	if err != nil {
		return err
	}
	u.Salt = salt
	u.Password = hash
	return db.Save(u).Error
}

// MakeAdmin is a simple wrapper around promoting a user to an administrator.
// If the user is already an admin, this function will return without modifying
// the database
func (u *User) MakeAdmin(db *gorm.DB) error {
	if !u.Admin {
		u.Admin = true
		return db.Save(&u).Error
	}
	return nil
}

type GetUserRequestBody struct {
	ID uint `json:"id" binding:"required"`
}

func GetUser(c *gin.Context) {
	var body GetUserRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf("Error parsing GetUser post body: %s", err.Error())
		return
	}

	db, err := GetDB(c)
	if err != nil {
		logrus.Errorf("Could not open database: %s", err.Error())
		DatabaseError(c)
		return
	}

	vc, err := VC(c, db)
	if err != nil {
		logrus.Errorf("Could not create VC: %s", err.Error())
		DatabaseError(c) // TODO tampered?
		return
	}

	var user User
	err = db.Where("id = ?", body.ID).First(&user).Error
	if err != nil {
		logrus.Errorf("Error finding user in database: %s", err.Error())
		DatabaseError(c)
		return
	}

	if !vc.CanPerformReadActionOnUser(db, &user) {
		UnauthorizedError(c)
		return
	}

	c.JSON(200, map[string]interface{}{
		"id":                   user.ID,
		"username":             user.Username,
		"name":                 user.Name,
		"email":                user.Email,
		"admin":                user.Admin,
		"enabled":              user.Enabled,
		"needs_password_reset": user.NeedsPasswordReset,
	})
}

type CreateUserRequestBody struct {
	Username           string `json:"username" binding:"required"`
	Password           string `json:"password" binding:"required"`
	Email              string `json:"email" binding:"required"`
	Admin              bool   `json:"admin"`
	NeedsPasswordReset bool   `json:"needs_password_reset"`
}

func CreateUser(c *gin.Context) {
	var body CreateUserRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf("Error parsing CreateUser post body: %s", err.Error())
		return
	}

	db, err := GetDB(c)
	if err != nil {
		logrus.Errorf("Could not open database: %s", err.Error())
		DatabaseError(c)
		return
	}

	vc, err := VC(c, db)
	if err != nil {
		logrus.Errorf("Could not create VC: %s", err.Error())
		DatabaseError(c)
		return
	}

	if !vc.IsAdmin() {
		UnauthorizedError(c)
		return
	}

	_, err = NewUser(db, body.Username, body.Password, body.Email, body.Admin, body.NeedsPasswordReset)
	if err != nil {
		logrus.Errorf("Error creating new user: %s", err.Error())
		DatabaseError(c)
		return
	}

	c.JSON(200, nil)
}

type ModifyUserRequestBody struct {
	ID       uint   `json:"id" binding:"required"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

func ModifyUser(c *gin.Context) {
	var body ModifyUserRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf("Error parsing ModifyUser post body: %s", err.Error())
		return
	}

	db, err := GetDB(c)
	if err != nil {
		logrus.Errorf("Could not open database: %s", err.Error())
		DatabaseError(c)
		return
	}

	vc, err := VC(c, db)
	if err != nil {
		logrus.Errorf("Could not create VC: %s", err.Error())
		DatabaseError(c)
		return
	}

	var user User
	err = db.Where("id = ?", body.ID).First(&user).Error
	if err != nil {
		logrus.Errorf("Error finding user in database: %s", err.Error())
		DatabaseError(c)
		return
	}

	if !vc.CanPerformWriteActionOnUser(db, &user) {
		UnauthorizedError(c)
		return
	}

	err = db.Save(&user).Error
	if err != nil {
		logrus.Errorf("Error updating user in database: %s", err.Error())
		DatabaseError(c)
		return
	}
	c.JSON(200, nil)
}

type DeleteUserRequestBody struct {
	ID uint `json:"id" binding:"required"`
}

func DeleteUser(c *gin.Context) {
	var body DeleteUserRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf("Error parsing DeleteUser post body: %s", err.Error())
		return
	}

	db, err := GetDB(c)
	if err != nil {
		logrus.Errorf("Could not open database: %s", err.Error())
		DatabaseError(c)
		return
	}

	vc, err := VC(c, db)
	if err != nil {
		logrus.Errorf("Could not create VC: %s", err.Error())
		DatabaseError(c)
		return
	}

	if !vc.IsAdmin() {
		UnauthorizedError(c)
		return
	}

	var user User
	err = db.Where("id = ?", body.ID).First(&user).Error
	if err != nil {
		logrus.Errorf("Error finding user in database: %s", err.Error())
		DatabaseError(c)
		return
	}

	err = db.Delete(&user).Error
	if err != nil {
		logrus.Errorf("Error deleting user from database: %s", err.Error())
		DatabaseError(c)
		return
	}
	c.JSON(200, nil)
}

type ResetPasswordRequestBody struct {
	ID             uint   `json:"id" binding:"required"`
	Password       string `json:"password" binding:"required"`
	PasswordConfim string `json:"password_confirm" binding:"required"`
}

func ResetUserPassword(c *gin.Context) {
	var body ResetPasswordRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf("Error parsing ResetPassword post body: %s", err.Error())
		return
	}

	if body.Password != body.PasswordConfim {
		c.JSON(406, map[string]interface{}{"error": "Passwords do not match"})
		return
	}

	db, err := GetDB(c)
	if err != nil {
		logrus.Errorf("Could not open database: %s", err.Error())
		DatabaseError(c)
		return
	}

	vc, err := VC(c, db)
	if err != nil {
		logrus.Errorf("Could not create VC: %s", err.Error())
		DatabaseError(c)
		return
	}
	var user User
	err = db.Where("id = ?", body.ID).First(&user).Error
	if err != nil {
		logrus.Errorf("Error finding user in database: %s", err.Error())
		DatabaseError(c)
		return
	}

	if !vc.CanPerformWriteActionOnUser(db, &user) {
		UnauthorizedError(c)
		return
	}

	err = user.SetPassword(db, body.Password)
	if err != nil {
		logrus.Errorf("Error setting user password: %s", err.Error())
		// xxx don't try to write to the db?
	}

	err = db.Save(&user).Error
	if err != nil {
		logrus.Errorf("Error updating user in database: %s", err.Error())
		DatabaseError(c)
		return
	}
	c.JSON(200, nil)
}

type SetUserAdminStateRequestBody struct {
	ID    uint `json:"id" binding:"required"`
	Admin bool `json:"admin" binding:"required"`
}

func SetUserAdminState(c *gin.Context) {
	var body SetUserAdminStateRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf("Error parsing SetUserAdminState post body: %s", err.Error())
		return
	}

	db, err := GetDB(c)
	if err != nil {
		logrus.Errorf("Could not open database: %s", err.Error())
		DatabaseError(c)
		return
	}

	vc, err := VC(c, db)
	if err != nil {
		logrus.Errorf("Could not create VC: %s", err.Error())
		DatabaseError(c)
		return
	}

	if !vc.IsAdmin() {
		UnauthorizedError(c)
		return
	}

	var user User
	err = db.Where("id = ?", body.ID).First(&user).Error
	if err != nil {
		logrus.Errorf("Error finding user in database: %s", err.Error())
		DatabaseError(c)
		return
	}

	user.Admin = body.Admin
	err = db.Save(&user).Error
	if err != nil {
		logrus.Errorf("Error updating user in database: %s", err.Error())
		DatabaseError(c)
		return
	}
	c.JSON(200, nil)
}

type SetUserEnabledStateRequestBody struct {
	ID      uint `json:"id" binding:"required"`
	Enabled bool `json:"enabled" binding:"required"`
}

func SetUserEnabledState(c *gin.Context) {
	var body SetUserEnabledStateRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf("Error parsing SetUserEnabledState post body: %s", err.Error())
		return
	}

	db, err := GetDB(c)
	if err != nil {
		logrus.Errorf("Could not open database: %s", err.Error())
		DatabaseError(c)
		return
	}

	vc, err := VC(c, db)
	if err != nil {
		logrus.Errorf("Could not create VC: %s", err.Error())
		DatabaseError(c)
		return
	}

	if !vc.IsAdmin() {
		UnauthorizedError(c)
		return
	}

	var user User
	err = db.Where("id = ?", body.ID).First(&user).Error
	if err != nil {
		logrus.Errorf("Error finding user in database: %s", err.Error())
		DatabaseError(c)
		return
	}

	user.Enabled = body.Enabled
	err = db.Save(&user).Error
	if err != nil {
		logrus.Errorf("Error updating user in database: %s", err.Error())
		DatabaseError(c)
		return
	}
	c.JSON(200, nil)
}
