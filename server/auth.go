package server

import (
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/kolide/kolide-ose/errors"
	"github.com/kolide/kolide-ose/kolide"
)

// ViewerContext is a struct which represents the ability for an execution
// context to participate in certain actions. Most often, a ViewerContext is
// associated with an application user, but a ViewerContext can represent a
// variety of other execution contexts as well (script, test, etc). The main
// purpose of a ViewerContext is to assist in the authorization of sensitive
// actions.
type ViewerContext struct {
	user *kolide.User
}

// IsAdmin indicates whether or not the current user can perform administrative
// actions.
func (vc *ViewerContext) IsAdmin() bool {
	if vc.user != nil {
		return vc.user.Admin && vc.user.Enabled
	}
	return false
}

// UserID is a helper that enables quick access to the user ID of the current
// user.
func (vc *ViewerContext) UserID() (uint, error) {
	if vc.user != nil {
		return vc.user.ID, nil
	}
	return 0, errors.New("Unauthorized", "No user set")
}

// CanPerformActions returns a bool indicating the current user's ability to
// perform the most basic actions on the site
func (vc *ViewerContext) CanPerformActions() bool {
	return vc.IsLoggedIn() && !vc.user.NeedsPasswordReset
}

// IsLoggedIn determines whether or not the current VC is attached to a user
// account
func (vc *ViewerContext) IsLoggedIn() bool {
	return vc.user != nil && vc.user.Enabled
}

// IsUserID returns true if the given user id the same as the user which is
// represented by this ViewerContext
func (vc *ViewerContext) IsUserID(id uint) bool {
	userID, err := vc.UserID()
	if err != nil {
		return false
	}
	if userID == id {
		return true
	}
	return false
}

// CanPerformWriteActionOnUser returns a bool indicating the current user's
// ability to perform write actions on the given user
func (vc *ViewerContext) CanPerformWriteActionOnUser(u *kolide.User) bool {
	return vc.CanPerformActions() && (vc.IsUserID(u.ID) || vc.IsAdmin())
}

// CanPerformReadActionsOnUser returns a bool indicating the current user's
// ability to perform read actions on the given user
func (vc *ViewerContext) CanPerformReadActionOnUser(u *kolide.User) bool {
	return vc.CanPerformActions() || (vc.IsLoggedIn() && vc.IsUserID(u.ID))
}

// GenerateVC generates a ViewerContext given a user struct
func GenerateVC(user *kolide.User) *ViewerContext {
	return &ViewerContext{
		user: user,
	}
}

// EmptyVC is a utility which generates an empty ViewerContext. This is often
// used to represent users which are not logged in.
func EmptyVC() *ViewerContext {
	return &ViewerContext{
		user: nil,
	}
}

// VC accepts a web request context and a database handler and attempts
// to parse a user's jwt token out of the active session, validate the token,
// and generate an appropriate ViewerContext given the data in the session.
func VC(c *gin.Context) *ViewerContext {
	sm := NewSessionManager(c)
	session, err := sm.Session()
	if err != nil {
		return EmptyVC()
	}
	return VCForID(GetDB(c), session.UserID)
}

func VCForID(db kolide.UserStore, id uint) *ViewerContext {
	// Generating a VC requires a user struct. Attempt to populate one using
	// the user id of the current session holder
	user, err := db.UserByID(id)
	if err != nil {
		return EmptyVC()
	}

	return GenerateVC(user)
}

////////////////////////////////////////////////////////////////////////////////
// Login and password utilities
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Authentication and authorization web endpoints
////////////////////////////////////////////////////////////////////////////////

// swagger:parameters Login
type LoginRequestBody struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// swagger:route POST /api/v1/kolide/login Login
//
// Login to the application
//
// This allows you to submit a set of credentials to the server and the
// server will attempt to validate the credentials against the backing
// database. If the credentials are valid, the response will issue the
// browser a new session cookie.
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
//       authenticated: no
//
//     Responses:
//       200: GetUserResponseBody
func Login(c *gin.Context) {
	var body LoginRequestBody
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	db := GetDB(c)

	user, err := db.User(body.Username)
	if err != nil {
		logrus.Debugf("User not found: %s", body.Username)
		UnauthorizedError(c)
		return
	}

	err = user.ValidatePassword(body.Password)
	if err != nil {
		logrus.Debugf("Invalid password for user: %s", body.Username)
		UnauthorizedError(c)
		return
	}

	sm := NewSessionManager(c)

	err = sm.MakeSessionForUserID(user.ID)
	if err != nil {
		errors.ReturnError(c, errors.DatabaseError(err))
		return
	}

	err = sm.Save()
	if err != nil {
		errors.ReturnError(c, errors.DatabaseError(err))
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

// swagger:route GET /api/v1/kolide/logout Logout
//
// Logout of the application
//
// This endpoint will delete the current session from the backend database
// and log the user out of the application
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
func Logout(c *gin.Context) {
	sm := NewSessionManager(c)

	err := sm.Destroy()
	if err != nil {
		errors.ReturnError(c, errors.DatabaseError(err))
		return
	}

	err = sm.Save()
	if err != nil {
		errors.ReturnError(c, errors.DatabaseError(err))
		return
	}

	c.JSON(200, nil)
}

////////////////////////////////////////////////////////////////////////////////
// Session management HTTP endpoints
////////////////////////////////////////////////////////////////////////////////

// swagger:parameters DeleteSession
type DeleteSessionRequestBody struct {
	SessionID uint `json:"session_id" validate:"required"`
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
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	db := GetDB(c)

	session, err := db.FindSessionByID(body.SessionID)
	if err != nil {
		errors.ReturnError(c, errors.DatabaseError(err))
		return
	}

	user, err := db.UserByID(session.UserID)
	if err != nil {
		errors.ReturnError(c, errors.DatabaseError(err))
		return
	}

	if !vc.CanPerformWriteActionOnUser(user) {
		UnauthorizedError(c)
		return
	}

	err = db.DestroySession(session)
	if err != nil {
		errors.ReturnError(c, errors.DatabaseError(err))
		return
	}

	c.JSON(http.StatusOK, nil)
}

// swagger:parameters DeleteSessionsForUser
type DeleteSessionsForUserRequestBody struct {
	ID uint `json:"id"`
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
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	db := GetDB(c)
	user, err := db.UserByID(body.ID)
	if err != nil {
		errors.ReturnError(c, errors.DatabaseError(err))
		return
	}

	if !vc.CanPerformWriteActionOnUser(user) {
		UnauthorizedError(c)
		return
	}

	err = db.DestroyAllSessionsForUser(user.ID)
	if err != nil {
		errors.ReturnError(c, errors.DatabaseError(err))
		return
	}

	c.JSON(http.StatusOK, nil)

}

// swagger:parameters GetInfoAboutSession
type GetInfoAboutSessionRequestBody struct {
	SessionKey string `json:"session_key" validate:"required"`
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
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	db := GetDB(c)
	session, err := db.FindSessionByKey(body.SessionKey)
	if err != nil {
		errors.ReturnError(c, errors.DatabaseError(err))
		return
	}

	user, err := db.UserByID(session.UserID)
	if err != nil {
		errors.ReturnError(c, errors.DatabaseError(err))
		return
	}

	if !vc.IsAdmin() && !vc.IsUserID(user.ID) {
		UnauthorizedError(c)
		return
	}

	c.JSON(http.StatusOK, &SessionInfoResponseBody{
		SessionID:  session.ID,
		UserID:     session.UserID,
		CreatedAt:  session.CreatedAt,
		AccessedAt: session.AccessedAt,
	})
}

// swagger:parameters GetInfoAboutSessionsForUser
type GetInfoAboutSessionsForUserRequestBody struct {
	ID uint `json:"id"`
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
	err := ParseAndValidateJSON(c, &body)
	if err != nil {
		errors.ReturnError(c, err)
		return
	}

	vc := VC(c)
	if !vc.CanPerformActions() {
		UnauthorizedError(c)
		return
	}

	db := GetDB(c)
	user, err := db.UserByID(body.ID)
	if err != nil {
		errors.ReturnError(c, errors.DatabaseError(err))
		return
	}

	if !vc.CanPerformWriteActionOnUser(user) {
		UnauthorizedError(c)
		return
	}

	sessions, err := db.FindAllSessionsForUser(user.ID)
	if err != nil {
		errors.ReturnError(c, errors.DatabaseError(err))
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

	c.JSON(http.StatusOK, &GetInfoAboutSessionsForUserResponseBody{
		Sessions: response,
	})
}
