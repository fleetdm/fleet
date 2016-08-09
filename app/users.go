package app

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
	ID                 uint `gorm:"primary_key"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
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

// swagger:parameters GetUser
type GetUserRequestBody struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// swagger:response GetUserResponseBody
type GetUserResponseBody struct {
	ID                 uint   `json:"id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	Name               string `json:"name"`
	Admin              bool   `json:"admin"`
	Enabled            bool   `json:"enabled"`
	NeedsPasswordReset bool   `json:"needs_password_reset"`
}

// swagger:route POST /api/v1/kolide/user GetUser
//
// Get information on a user
//
// Using this API will allow the requester to inspect and get info on
// other users in the application.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetUserResponseBody
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

// swagger:parameters CreateUser
type CreateUserRequestBody struct {
	Username           string `json:"username" binding:"required"`
	Password           string `json:"password" binding:"required"`
	Email              string `json:"email" binding:"required"`
	Admin              bool   `json:"admin"`
	NeedsPasswordReset bool   `json:"needs_password_reset"`
}

// swagger:route PUT /api/v1/kolide/user CreateUser
//
// Create a new user
//
// Using this API will allow the requester to create a new user with the ability
// to control various user settings
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetUserResponseBody
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

// swagger:parameters ModifyUser
type ModifyUserRequestBody struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

// swagger:route PATCH /api/v1/kolide/user ModifyUser
//
// Update a user's basic information and settings
//
// Using this API will allow the requester to update a user's basic settings.
// Note that updating administrative settings are not exposed via this endpoint
// as this is primarily intended to be used by users to update their own
// settings.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetUserResponseBody
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

// swagger:parameters DeleteUser
type DeleteUserRequestBody struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// swagger:route DELETE /api/v1/kolide/user DeleteUser
//
// Delete a user account
//
// Using this API will allow the requester to permanently delete a user account.
// This endpoint enforces that the requester is an admin.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: nil
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

// swagger:parameters ChangeUserPassword
type ChangePasswordRequestBody struct {
	ID                uint   `json:"id"`
	Username          string `json:"username"`
	CurrentPassword   string `json:"current_password"`
	NewPassword       string `json:"new_password" binding:"required"`
	NewPasswordConfim string `json:"new_password_confirm" binding:"required"`
}

// swagger:route PATCH /api/v1/kolide/user/password ChangeUserPassword
//
// Change a user's password
//
// Using this API will allow the requester to change their password. Users
// should include their own user id as the "id" paramater and/or their own
// username as the "username" parameter. Admins can change the passords for
// other users by defining their ID or username.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetUserResponseBody
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

// swagger:parameters SetUserAdminState
type SetUserAdminStateRequestBody struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Admin    bool   `json:"admin"`
}

// swagger:route PATCH /api/v1/kolide/user/admin SetUserAdminState
//
// Modify a user's admin settings
//
// This endpoint allows an existing admin to promote a non-admin to admin or
// demote a current admin to non-admin.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetUserResponseBody
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

// swagger:parameters SetUserEnabledState
type SetUserEnabledStateRequestBody struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Enabled  bool   `json:"enabled"`
}

// swagger:route PATCH /api/v1/kolide/user/enabled SetUserEnabledState
//
// Enable or disable a user.
//
// This endpoint allows an existing admin to enable a disabled user or
// disable an enabled user.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetUserResponseBody
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

// swagger:parameters DeleteSession
type DeleteSessionRequestBody struct {
	SessionID uint `json:"session_id" binding:"required"`
}

// swagger:route DELETE /api/v1/kolide/session DeleteSession
//
// Delete a specific session, as specified by the session's ID.
//
// This API allows for the requester to delete a specific session. Note that the
// API expects the session ID as the parameter, not the session key.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: nil
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
	user := &User{ID: session.UserID}
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

// swagger:parameters DeleteSessionsForUser
type DeleteSessionsForUserRequestBody struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// swagger:route DELETE /api/v1/kolide/user/sessions DeleteSessionsForUser
//
// Delete all of a user's active sessions
//
// This API allows an admin to delete all active sessions that are known to
// belong to a specific user. This effectively logs out the user on all
// devices.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: nil
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

// swagger:parameters GetInfoAboutSession
type GetInfoAboutSessionRequestBody struct {
	SessionKey string `json:"session_key" binding:"required"`
}

// swagger:response SessionInfoResponseBody
type SessionInfoResponseBody struct {
	SessionID  uint      `json:"session_id"`
	UserID     uint      `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
	AccessedAt time.Time `json:"created_at"`
}

// swagger:route POST /api/v1/kolide/session GetInfoAboutSession
//
// Get information on a session, given a session key.
//
// Using this API will allow the requester to inspect and get info on
// an active session, given the session key.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: SessionInfoResponseBody
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

// swagger:parameters GetInfoAboutSessionsForUser
type GetInfoAboutSessionsForUserRequestBody struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// swagger:response GetInfoAboutSessionsForUserResponseBody
type GetInfoAboutSessionsForUserResponseBody struct {
	Sessions []SessionInfoResponseBody `json:"sessions"`
}

// swagger:route POST /api/v1/kolide/user/sessions GetInfoAboutSessionsForUser
//
// Get information on a user's sessions
//
// Using this API will allow an admin to inspect and get info on all of a user's
// active session.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: https
//
//     Security:
//       authenticated: yes
//
//     Responses:
//       200: GetInfoAboutSessionsForUserResponseBody
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

	var response []SessionInfoResponseBody
	for _, session := range sessions {
		response = append(response, SessionInfoResponseBody{
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
