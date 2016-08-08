package main

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/sessions"
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
	saltAndPass := []byte(fmt.Sprintf("%s%s", password, u.Salt))
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

////////////////////////////////////////////////////////////////////////////////
// User management web endpoints
////////////////////////////////////////////////////////////////////////////////

type GetUserRequestBody struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type GetUserResponseBody struct {
	ID                 uint   `json:"id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	Name               string `json:"name"`
	Admin              bool   `json:"admin"`
	Enabled            bool   `json:"enabled"`
	NeedsPasswordReset bool   `json:"needs_password_reset"`
}

func GetUser(c *gin.Context) {
	var body GetUserRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf("Error parsing GetUser post body: %s", err.Error())
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	db := GetDB(c)
	var user User
	user.ID = body.ID
	user.Username = body.Username
	err = db.Where(&user).First(&user).Error
	if err != nil {
		DatabaseError(c)
		return
	}

	if !vc.CanPerformReadActionOnUser(&user) {
		UnauthorizedError(c)
		return
	}

	c.JSON(200, GetUserResponseBody{
		ID:                 user.ID,
		Username:           user.Username,
		Name:               user.Name,
		Email:              user.Email,
		Admin:              user.Admin,
		Enabled:            user.Enabled,
		NeedsPasswordReset: user.NeedsPasswordReset,
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

	vc := VC(c)
	if !vc.IsAdmin() {
		UnauthorizedError(c)
		return
	}

	db := GetDB(c)
	user, err := NewUser(db, body.Username, body.Password, body.Email, body.Admin, body.NeedsPasswordReset)
	if err != nil {
		logrus.Errorf("Error creating new user: %s", err.Error())
		DatabaseError(c)
		return
	}

	c.JSON(200, GetUserResponseBody{
		ID:                 user.ID,
		Username:           user.Username,
		Name:               user.Name,
		Email:              user.Email,
		Admin:              user.Admin,
		Enabled:            user.Enabled,
		NeedsPasswordReset: user.NeedsPasswordReset,
	})
}

type ModifyUserRequestBody struct {
	ID       uint   `json:"id"`
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

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	var user User
	user.ID = body.ID
	user.Username = body.Username

	db := GetDB(c)
	err = db.Where(&user).First(&user).Error
	if err != nil {
		DatabaseError(c)
		return
	}

	if !vc.CanPerformWriteActionOnUser(&user) {
		UnauthorizedError(c)
		return
	}

	if body.Name != "" {
		user.Name = body.Name
	}
	if body.Email != "" {
		user.Email = body.Email
	}
	err = db.Save(&user).Error
	if err != nil {
		logrus.Errorf("Error updating user in database: %s", err.Error())
		DatabaseError(c)
		return
	}
	c.JSON(200, GetUserResponseBody{
		ID:                 user.ID,
		Username:           user.Username,
		Name:               user.Name,
		Email:              user.Email,
		Admin:              user.Admin,
		Enabled:            user.Enabled,
		NeedsPasswordReset: user.NeedsPasswordReset,
	})
}

type DeleteUserRequestBody struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

func DeleteUser(c *gin.Context) {
	var body DeleteUserRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf("Error parsing DeleteUser post body: %s", err.Error())
		return
	}

	vc := VC(c)
	if !vc.IsAdmin() {
		UnauthorizedError(c)
		return
	}

	db := GetDB(c)
	var user User
	user.ID = body.ID
	user.Username = body.Username
	err = db.Where(&user).First(&user).Error
	if err != nil {
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
	ID             uint   `json:"id"`
	Username       string `json:"username"`
	Password       string `json:"password" binding:"required"`
	PasswordConfim string `json:"password_confirm" binding:"required"`
}

type ChangePasswordRequestBody struct {
	ID                uint   `json:"id"`
	Username          string `json:"username"`
	CurrentPassword   string `json:"current_password"`
	NewPassword       string `json:"new_password" binding:"required"`
	NewPasswordConfim string `json:"new_password_confirm" binding:"required"`
}

func ChangeUserPassword(c *gin.Context) {
	var body ChangePasswordRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf("Error parsing ResetPassword post body: %s", err.Error())
		return
	}

	if body.NewPassword != body.NewPasswordConfim {
		c.JSON(406, map[string]interface{}{"error": "Passwords do not match"})
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	db := GetDB(c)
	var user User
	user.ID = body.ID
	user.Username = body.Username
	err = db.Where(&user).First(&user).Error
	if err != nil {
		DatabaseError(c)
		return
	}

	if !vc.IsAdmin() {
		if !vc.IsUserID(user.ID) {
			UnauthorizedError(c)
			return
		}
		if user.ValidatePassword(body.CurrentPassword) != nil {
			UnauthorizedError(c)
			return
		}
	}

	err = user.SetPassword(db, body.NewPassword)
	if err != nil {
		logrus.Errorf("Error setting user password: %s", err.Error())
		DatabaseError(c) // probably not this
		return
	}

	err = db.Save(&user).Error
	if err != nil {
		logrus.Errorf("Error updating user in database: %s", err.Error())
		DatabaseError(c)
		return
	}
	c.JSON(200, GetUserResponseBody{
		ID:                 user.ID,
		Username:           user.Username,
		Name:               user.Name,
		Email:              user.Email,
		Admin:              user.Admin,
		Enabled:            user.Enabled,
		NeedsPasswordReset: user.NeedsPasswordReset,
	})
}

type SetUserAdminStateRequestBody struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Admin    bool   `json:"admin"`
}

func SetUserAdminState(c *gin.Context) {
	var body SetUserAdminStateRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf("Error parsing SetUserAdminState post body: %s", err.Error())
		return
	}

	vc := VC(c)
	if !vc.IsAdmin() {
		UnauthorizedError(c)
		return
	}

	db := GetDB(c)
	var user User
	user.ID = body.ID
	user.Username = body.Username
	err = db.Where(&user).First(&user).Error
	if err != nil {
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
	c.JSON(200, GetUserResponseBody{
		ID:                 user.ID,
		Username:           user.Username,
		Name:               user.Name,
		Email:              user.Email,
		Admin:              user.Admin,
		Enabled:            user.Enabled,
		NeedsPasswordReset: user.NeedsPasswordReset,
	})
}

type SetUserEnabledStateRequestBody struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Enabled  bool   `json:"enabled"`
}

func SetUserEnabledState(c *gin.Context) {
	var body SetUserEnabledStateRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf("Error parsing SetUserEnabledState post body: %s", err.Error())
		return
	}

	vc := VC(c)
	if !vc.IsAdmin() {
		UnauthorizedError(c)
		return
	}

	db := GetDB(c)
	var user User
	user.ID = body.ID
	user.Username = body.Username
	err = db.Where(&user).First(&user).Error
	if err != nil {
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
	c.JSON(200, GetUserResponseBody{
		ID:                 user.ID,
		Username:           user.Username,
		Name:               user.Name,
		Email:              user.Email,
		Admin:              user.Admin,
		Enabled:            user.Enabled,
		NeedsPasswordReset: user.NeedsPasswordReset,
	})
}

///////////////////////////////////////////////////////////////////////////////
// Session management HTTP endpoints
////////////////////////////////////////////////////////////////////////////////

// Setting the session backend via a middleware
func SessionBackendMiddleware(c *gin.Context) {
	db := GetDB(c)
	c.Set("SessionBackend", &sessions.GormSessionBackend{DB: db})
	c.Next()
}

// Get the database connection from the context, or panic
func GetSessionBackend(c *gin.Context) sessions.SessionBackend {
	return c.MustGet("SessionBackend").(sessions.SessionBackend)
}

////////////////////////////////////////////////////////////////////////////////
// Session management HTTP endpoints
////////////////////////////////////////////////////////////////////////////////

type DeleteSessionRequestBody struct {
	SessionID uint `json:"session_id" binding:"required"`
}

func DeleteSession(c *gin.Context) {
	var body DeleteSessionRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf(err.Error())
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	sb := GetSessionBackend(c)

	session, err := sb.FindID(body.SessionID)
	if err != nil {

	}

	db := GetDB(c)
	user := &User{
		BaseModel: BaseModel{
			ID: session.UserID,
		},
	}
	err = db.Where(user).First(user).Error
	if err != nil {
		DatabaseError(c)
		return
	}

	if !vc.CanPerformWriteActionOnUser(user) {
		UnauthorizedError(c)
		return
	}

	err = sb.Destroy(session)
	if err != nil {
		DatabaseError(c)
		return
	}

	c.JSON(200, nil)
}

type DeleteSessionsForUserRequestBody struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

func DeleteSessionsForUser(c *gin.Context) {
	var body DeleteSessionsForUserRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf(err.Error())
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	db := GetDB(c)
	var user User
	user.ID = body.ID
	user.Username = body.Username
	err = db.Where(&user).First(&user).Error
	if err != nil {
		DatabaseError(c)
		return
	}

	if !vc.CanPerformWriteActionOnUser(&user) {
		UnauthorizedError(c)
		return
	}

	sb := GetSessionBackend(c)
	err = sb.DestroyAllForUser(user.ID)
	if err != nil {
		DatabaseError(c)
		return
	}

	c.JSON(200, nil)

}

type GetInfoAboutSessionRequestBody struct {
	SessionKey string `json:"session_key" binding:"required"`
}

type SessionInfoResponseBody struct {
	SessionID  uint      `json:"session_id"`
	UserID     uint      `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
	AccessedAt time.Time `json:"created_at"`
}

func GetInfoAboutSession(c *gin.Context) {
	var body GetInfoAboutSessionRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf(err.Error())
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	sb := GetSessionBackend(c)
	session, err := sb.FindKey(body.SessionKey)
	if err != nil {
		DatabaseError(c)
		return
	}

	db := GetDB(c)
	var user User
	user.ID = session.UserID
	err = db.Where(&user).First(&user).Error
	if err != nil {
		DatabaseError(c)
		return
	}

	if !vc.IsAdmin() && !vc.IsUserID(user.ID) {
		UnauthorizedError(c)
		return
	}

	c.JSON(200, &SessionInfoResponseBody{
		SessionID:  session.ID,
		UserID:     session.UserID,
		CreatedAt:  session.CreatedAt,
		AccessedAt: session.AccessedAt,
	})
}

type GetInfoAboutSessionsForUserRequestBody struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type GetInfoAboutSessionsForUserResponseBody struct {
	Sessions []*SessionInfoResponseBody `json:"sessions"`
}

func GetInfoAboutSessionsForUser(c *gin.Context) {
	var body GetInfoAboutSessionsForUserRequestBody
	err := c.BindJSON(&body)
	if err != nil {
		logrus.Errorf(err.Error())
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	db := GetDB(c)
	var user User
	user.ID = body.ID
	user.Username = body.Username
	err = db.Where(&user).First(&user).Error
	if err != nil {
		DatabaseError(c)
		return
	}

	if !vc.IsAdmin() && !vc.IsUserID(user.ID) {
		UnauthorizedError(c)
		return
	}

	sb := GetSessionBackend(c)
	sessions, err := sb.FindAllForUser(user.ID)
	if err != nil {
		DatabaseError(c)
		return
	}

	var response []*SessionInfoResponseBody
	for _, session := range sessions {
		response = append(response, &SessionInfoResponseBody{
			SessionID:  session.ID,
			UserID:     session.UserID,
			CreatedAt:  session.CreatedAt,
			AccessedAt: session.AccessedAt,
		})
	}

	c.JSON(200, &GetInfoAboutSessionsForUserResponseBody{
		Sessions: response,
	})
}
