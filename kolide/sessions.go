package kolide

import (
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/kolide/kolide-ose/errors"
	"github.com/spf13/viper"
)

const publicErrorMessage string = "Session error"

var (
	// An error returned by SessionStore.Get() if no session record was found
	// in the database
	ErrNoActiveSession = errors.New(publicErrorMessage, "Active session is not present in the database")

	// An error returned by SessionStore methods when no session object has
	// been created yet but the requested action requires one
	ErrSessionNotCreated = errors.New(publicErrorMessage, "The session has not been created")

	// An error returned by SessionStore.Get() when a session is requested but
	// it has expired
	ErrSessionExpired = errors.New(publicErrorMessage, "The session has expired")

	// An error returned by SessionStore which indicates that the token
	// or it's content were malformed
	ErrSessionMalformed = errors.New(publicErrorMessage, "The session token was malformed")
)

// SessionStore is the abstract interface that all session backends must
// conform to.
type SessionStore interface {
	// Given a session key, find and return a session object or an error if one
	// could not be found for the given key
	FindSessionByKey(key string) (*Session, error)

	// Given a session id, find and return a session object or an error if one
	// could not be found for the given id
	FindSessionByID(id uint) (*Session, error)

	// Find all of the active sessions for a given user
	FindAllSessionsForUser(id uint) ([]*Session, error)

	// Create a session object tied to the given user ID
	CreateSessionForUserID(userID uint) (*Session, error)

	// Destroy the currently tracked session
	DestroySession(session *Session) error

	// Destroy all of the sessions for a given user
	DestroyAllSessionsForUser(id uint) error

	// Mark the currently tracked session as access to extend expiration
	MarkSessionAccessed(session *Session) error
}

// Session is the model object which represents what an active session is
type Session struct {
	ID         uint `gorm:"primary_key"`
	CreatedAt  time.Time
	AccessedAt time.Time
	UserID     uint   `gorm:"not null"`
	Key        string `gorm:"not null;unique_index:idx_session_unique_key"`
}

////////////////////////////////////////////////////////////////////////////////
// Managing sessions
////////////////////////////////////////////////////////////////////////////////

// SessionManager is a management object which helps with the administration of
// sessions within the application. Use NewSessionManager to create an instance
type SessionManager struct {
	Store   SessionStore
	Request *http.Request
	Writer  http.ResponseWriter
	session *Session
}

func (sm *SessionManager) Session() (*Session, error) {
	if sm.session == nil {
		cookie, err := sm.Request.Cookie(viper.GetString("session.cookie_name"))
		if err != nil {
			switch err {
			case http.ErrNoCookie:
				// No cookie was set
				return nil, err
			default:
				// Something went wrong and the cookie may or may not be set
				logrus.Errorf("Couldn't get cookie: %s", err.Error())
				return nil, ErrSessionMalformed
			}
		}

		token, err := ParseJWT(cookie.Value)
		if err != nil {
			logrus.Errorf("Couldn't parse JWT token string from cookie: %s", err.Error())
			return nil, ErrSessionMalformed
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			logrus.Error("Could not parse the claims from the JWT token")
			return nil, ErrSessionMalformed
		}

		sessionKeyClaim, ok := claims["session_key"]
		if !ok {
			logrus.Warn("JWT did not have session_key claim")
			return nil, ErrSessionMalformed
		}

		sessionKey, ok := sessionKeyClaim.(string)
		if !ok {
			logrus.Warn("JWT session_key claim was not a string")
			return nil, ErrSessionMalformed
		}

		session, err := sm.Store.FindSessionByKey(sessionKey)
		if err != nil {
			switch err {
			case ErrNoActiveSession:
				// If the code path got this far, it's likely that the user was logged
				// in some time in the past, but their session has been expired since
				// their last usage of the application
				return nil, err
			default:
				logrus.Errorf("Couldn't call Get on backend object: %s", err.Error())
				return nil, err
			}
		}
		sm.session = session
	}
	return sm.session, nil

}

// MakeSessionForUserID creates a session in the database for a given user id.
// You must call Save() after calling this.
func (sm *SessionManager) MakeSessionForUserID(id uint) error {
	session, err := sm.Store.CreateSessionForUserID(id)
	if err != nil {
		return errors.New("Session error", "Error creating session via the store")
	}
	sm.session = session
	return nil
}

// Save writes the current session to a token and delivers the token as a cookie
// to the user. Save must be called after every write action on this struct
// (MakeSessionForUser, Destroy, etc.)
func (sm *SessionManager) Save() error {
	var token string
	var err error
	if sm.session != nil {
		token, err = GenerateJWT(sm.session.Key)
		if err != nil {
			return err
		}
	}

	// TODO: set proper flags on cookie for maximum security
	cookieName := viper.GetString("session.cookie_name")
	if cookieName == "" {
		cookieName = "KolideSession"
	}

	cookie := &http.Cookie{
		Name:  cookieName,
		Value: token,
	}
	http.SetCookie(sm.Writer, cookie)

	return err
}

// Destroy deletes the active session from the database and erases the session
// instance from this object's access. You must call Save() after calling this.
func (sm *SessionManager) Destroy() error {
	_, err := sm.Session()
	if err != nil {
		return err
	}

	if sm.Store != nil && sm.session != nil {
		err := sm.Store.DestroySession(sm.session)
		if err != nil {
			return err
		}
	}

	sm.session = nil

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// JSON Web Tokens
////////////////////////////////////////////////////////////////////////////////

// Given a session key create a JWT to be delivered to the client
func GenerateJWT(sessionKey string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"session_key": sessionKey,
	})

	return token.SignedString([]byte(viper.GetString("auth.jwt_key")))
}

// ParseJWT attempts to parse a JWT token in serialized string form into a
// JWT token in a deserialized jwt.Token struct.
func ParseJWT(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		method, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok || method != jwt.SigningMethodHS256 {
			return nil, errors.New(publicErrorMessage, "Unexpected signing method")
		}
		return []byte(viper.GetString("auth.jwt_key")), nil
	})
}
