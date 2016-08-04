package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

var (
	// An error returned by SessionBackend.Get() if no session record was found
	// in the database
	ErrNoActiveSession = errors.New("Active session is not present in the database")

	// An error returned by SessionBackend methods when no session object has
	// been created yet but the requested action requires one
	ErrSessionNotCreated = errors.New("The session has not been created")

	// An error returned by SessionBackend.Get() when a session is requested but
	// it has expired
	ErrSessionExpired = errors.New("The session has expired")
)

const (
	// The name of the session cookie
	CookieName = "KolideSession"
)

// Session is the model object which represents what an active session is
type Session struct {
	BaseModel
	UserID     uint   `gorm:"not null"`
	Key        string `gorm:"not null;unique_index:idx_session_unique_key"`
	AccessedAt time.Time
}

////////////////////////////////////////////////////////////////////////////////
// Managing sessions
////////////////////////////////////////////////////////////////////////////////

// SessionManager is a management object which helps with the administration of
// sessions within the application. Use NewSessionManager to create an instance
type SessionManager struct {
	backend SessionBackend
	request *http.Request
	writer  http.ResponseWriter
	session *Session
	vc      *ViewerContext
	db      *gorm.DB
}

// NewSessionManager allows you to get a SessionManager instance for a given
// web request. Unless you're interacting with login, logout, or core auth
// code, this should be abstracted by the ViewerContext pattern.
func NewSessionManager(c *gin.Context) *SessionManager {
	return &SessionManager{
		request: c.Request,
		backend: GetSessionBackend(c),
		writer:  c.Writer,
		db:      GetDB(c),
	}
}

// Get the ViewerContext instance for a user represented by the active session
func (sm *SessionManager) VC() *ViewerContext {
	if sm.session == nil {
		cookie, err := sm.request.Cookie(CookieName)
		if err != nil {
			switch err {
			case http.ErrNoCookie:
				// No cookie was set
				return EmptyVC()
			default:
				// Something went wrong and the cookie may or may not be set
				logrus.Errorf("Couldn't get cookie: %s", err.Error())
				return EmptyVC()
			}
		}

		token, err := ParseJWT(cookie.Value)
		if err != nil {
			logrus.Errorf("Couldn't parse JWT token string from cookie: %s", err.Error())
			return EmptyVC()
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			logrus.Error("Could not parse the claims from the JWT token")
			return EmptyVC()
		}

		sessionKeyClaim, ok := claims["session_key"]
		if !ok {
			logrus.Warn("JWT did not have session_key claim")
			return EmptyVC()
		}

		sessionKey, ok := sessionKeyClaim.(string)
		if !ok {
			logrus.Warn("JWT session_key claim was not a string")
			return EmptyVC()
		}

		session, err := sm.backend.FindKey(sessionKey)
		if err != nil {
			switch err {
			case ErrNoActiveSession:
				// If the code path got this far, it's likely that the user was logged
				// in some time in the past, but their session has been expired since
				// their last usage of the application
				return EmptyVC()
			default:
				logrus.Errorf("Couldn't call Get on backend object: %s", err.Error())
				return EmptyVC()
			}
		}
		sm.session = session
	}

	if sm.vc == nil {
		// Generating a VC requires a user struct. Attempt to populate one using
		// the user id of the current session holder
		user := &User{BaseModel: BaseModel{ID: sm.session.UserID}}
		err := sm.db.Where(user).First(user).Error
		if err != nil {
			return EmptyVC()
		}

		sm.vc = GenerateVC(user)
	}

	return sm.vc
}

// MakeSessionForUserID creates a session in the database for a given user id.
// You must call Save() after calling this.
func (sm *SessionManager) MakeSessionForUserID(id uint) error {
	session, err := sm.backend.Create(id)
	if err != nil {
		return err
	}
	sm.session = session
	return nil
}

// MakeSessionForUserID creates a session in the database for a given user
// You must call Save() after calling this.
func (sm *SessionManager) MakeSessionForUser(u *User) error {
	return sm.MakeSessionForUserID(u.ID)
}

// Save writes the current session to a token and delivers the token as a cookie
// to the user. Save must be called after every write action on this struct
// (MakeSessionForUser, Destroy, etc.)
func (sm *SessionManager) Save() error {
	token, err := GenerateJWT(sm.session.Key)
	if err != nil {
		return err
	}

	// TODO: set proper flags on cookie for maximum security
	http.SetCookie(sm.writer, &http.Cookie{
		Name:  CookieName,
		Value: token,
	})

	return nil
}

// Destroy deletes the active session from the database and erases the session
// instance from this object's access. You must call Save() after calling this.
func (sm *SessionManager) Destroy() error {
	if sm.backend != nil {
		err := sm.backend.Destroy(sm.session)
		if err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Session Backend API
////////////////////////////////////////////////////////////////////////////////

// SessionBackend is the abstract interface that all session backends must
// conform to. SessionBackend instances are only expected to exist within the
// context of a single request.
type SessionBackend interface {
	// Given a session key, find and return a session object or an error if one
	// could not be found for the given key
	FindKey(key string) (*Session, error)

	// Given a session id, find and return a session object or an error if one
	// could not be found for the given id
	FindID(id uint) (*Session, error)

	// Find all of the active sessions for a given user
	FindAllForUser(id uint) ([]*Session, error)

	// Create a session object tied to the given user ID
	Create(userID uint) (*Session, error)

	// Destroy the currently tracked session
	Destroy(session *Session) error

	// Destroy all of the sessions for a given user
	DestroyAllForUser(id uint) error

	// Mark the currently tracked session as access to extend expiration
	MarkAccessed(session *Session) error
}

////////////////////////////////////////////////////////////////////////////////
// Session Backend Plugins
////////////////////////////////////////////////////////////////////////////////

// GormSessionBackend stores sessions using a pre-instantiated gorm database
// object
type GormSessionBackend struct {
	db *gorm.DB
}

func (s *GormSessionBackend) validate(session *Session) error {
	if time.Since(session.AccessedAt).Seconds() >= config.App.SessionExpirationSeconds {
		err := s.db.Delete(session).Error
		if err != nil {
			return err
		}
		return ErrSessionExpired
	}

	err := s.MarkAccessed(session)
	if err != nil {
		return err
	}

	return nil
}

func (s *GormSessionBackend) FindID(id uint) (*Session, error) {
	session := &Session{
		BaseModel: BaseModel{
			ID: id,
		},
	}

	err := s.db.Where(session).First(session).Error
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, ErrNoActiveSession
		default:
			return nil, err
		}
	}

	err = s.validate(session)
	if err != nil {
		return nil, err
	}

	return session, nil

}

func (s *GormSessionBackend) FindKey(key string) (*Session, error) {
	session := &Session{
		Key: key,
	}

	err := s.db.Where(session).First(session).Error
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, ErrNoActiveSession
		default:
			return nil, err
		}
	}

	err = s.validate(session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *GormSessionBackend) FindAllForUser(id uint) ([]*Session, error) {
	var sessions []*Session
	err := s.db.Where("user_id = ?", id).Find(&sessions).Error
	return sessions, err
}

func (s *GormSessionBackend) Create(userID uint) (*Session, error) {
	key, err := generateRandomText(config.App.SessionKeySize)
	if err != nil {
		return nil, err
	}

	session := &Session{
		UserID: userID,
		Key:    key,
	}

	err = s.db.Create(session).Error
	if err != nil {
		return nil, err
	}

	err = s.MarkAccessed(session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *GormSessionBackend) Destroy(session *Session) error {
	err := s.db.Delete(session).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *GormSessionBackend) DestroyAllForUser(id uint) error {
	return s.db.Delete(&Session{}, "user_id = ?", id).Error
}

func (s *GormSessionBackend) MarkAccessed(session *Session) error {
	session.AccessedAt = time.Now().UTC()
	return s.db.Save(session).Error
}

////////////////////////////////////////////////////////////////////////////////
// Session management HTTP endpoints
////////////////////////////////////////////////////////////////////////////////

// Setting the session backend via a middleware
func SessionBackendMiddleware(c *gin.Context) {
	db := GetDB(c)
	c.Set("SessionBackend", &GormSessionBackend{db})
	c.Next()
}

// Get the database connection from the context, or panic
func GetSessionBackend(c *gin.Context) SessionBackend {
	return c.MustGet("SessionBackend").(SessionBackend)
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
	err = db.Delete(&Session{}, "user_id = ?", user.ID).Error
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
