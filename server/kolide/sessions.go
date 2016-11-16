package kolide

import (
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/kolide/kolide-ose/server/errors"
	"golang.org/x/net/context"
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
	SessionByKey(key string) (*Session, error)

	// Given a session id, find and return a session object or an error if one
	// could not be found for the given id
	SessionByID(id uint) (*Session, error)

	// Find all of the active sessions for a given user
	ListSessionsForUser(id uint) ([]*Session, error)

	// Store a new session struct
	NewSession(session *Session) (*Session, error)

	// Destroy the currently tracked session
	DestroySession(session *Session) error

	// Destroy all of the sessions for a given user
	DestroyAllSessionsForUser(id uint) error

	// Mark the currently tracked session as access to extend expiration
	MarkSessionAccessed(session *Session) error
}

type SessionService interface {
	Login(ctx context.Context, username, password string) (user *User, token string, err error)
	Logout(ctx context.Context) (err error)
	DestroySession(ctx context.Context) (err error)
	GetInfoAboutSessionsForUser(ctx context.Context, id uint) (sessions []*Session, err error)
	DeleteSessionsForUser(ctx context.Context, id uint) (err error)
	GetInfoAboutSession(ctx context.Context, id uint) (session *Session, err error)
	GetSessionByKey(ctx context.Context, key string) (session *Session, err error)
	DeleteSession(ctx context.Context, id uint) (err error)
}

// Session is the model object which represents what an active session is
type Session struct {
	CreateTimestamp
	ID         uint
	AccessedAt time.Time `db:"accessed_at"`
	UserID     uint      `db:"user_id"`
	Key        string
}

////////////////////////////////////////////////////////////////////////////////
// JSON Web Tokens
////////////////////////////////////////////////////////////////////////////////

// Given a session key create a JWT to be delivered to the client
func GenerateJWT(sessionKey, jwtKey string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"session_key": sessionKey,
	})

	return token.SignedString([]byte(jwtKey))
}

// ParseJWT attempts to parse a JWT token in serialized string form into a
// JWT token in a deserialized jwt.Token struct.
func ParseJWT(token, jwtKey string) (*jwt.Token, error) {
	return jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		method, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok || method != jwt.SigningMethodHS256 {
			return nil, errors.New(publicErrorMessage, "Unexpected signing method")
		}
		return []byte(jwtKey), nil
	})
}
