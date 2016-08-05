package sessions

import (
	"errors"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
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

	// An error returned by SessionBackend which indicates that the token
	// or it's content were malformed
	ErrSessionMalformed = errors.New("The session token was malformed")
)

var (
	// The name of the session cookie
	CookieName = "Session"

	// The key to be used to sign and verify JWTs
	jwtKey = ""

	// The amount of random data, in bytes, which will be used to create each
	// session key
	SessionKeySize = 64

	// The amount of seconds that will pass before an inactive user is logged out
	Lifespan = float64(60 * 60 * 24 * 90)
)

// Session is the model object which represents what an active session is
type Session struct {
	ID         uint `gorm:"primary_key"`
	CreatedAt  time.Time
	AccessedAt time.Time
	UserID     uint   `gorm:"not null"`
	Key        string `gorm:"not null;unique_index:idx_session_unique_key"`
}

////////////////////////////////////////////////////////////////////////////////
// Configuring the library
////////////////////////////////////////////////////////////////////////////////

type SessionConfiguration struct {
	CookieName     string
	JWTKey         string
	SessionKeySize int
	Lifespan       float64
}

func Configure(s *SessionConfiguration) {
	CookieName = s.CookieName
	jwtKey = s.JWTKey
	SessionKeySize = s.SessionKeySize
	Lifespan = s.Lifespan
}

// Set the name of the cookie
func SetCookieName(name string) {
	CookieName = name
}

////////////////////////////////////////////////////////////////////////////////
// Managing sessions
////////////////////////////////////////////////////////////////////////////////

// SessionManager is a management object which helps with the administration of
// sessions within the application. Use NewSessionManager to create an instance
type SessionManager struct {
	Backend SessionBackend
	Request *http.Request
	Writer  http.ResponseWriter
	session *Session
}

func (sm *SessionManager) Session() (*Session, error) {
	if sm.session == nil {
		cookie, err := sm.Request.Cookie(CookieName)
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

		session, err := sm.Backend.FindKey(sessionKey)
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
	session, err := sm.Backend.Create(id)
	if err != nil {
		return err
	}
	sm.session = session
	return nil
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
	http.SetCookie(sm.Writer, &http.Cookie{
		Name:  CookieName,
		Value: token,
	})

	return nil
}

// Destroy deletes the active session from the database and erases the session
// instance from this object's access. You must call Save() after calling this.
func (sm *SessionManager) Destroy() error {
	if sm.Backend != nil {
		err := sm.Backend.Destroy(sm.session)
		if err != nil {
			return err
		}
	}
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

	return token.SignedString([]byte(jwtKey))
}

// ParseJWT attempts to parse a JWT token in serialized string form into a
// JWT token in a deserialized jwt.Token struct.
func ParseJWT(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		method, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok || method != jwt.SigningMethodHS256 {
			return nil, errors.New("Unexpected signing method")
		}
		return []byte(jwtKey), nil
	})
}
