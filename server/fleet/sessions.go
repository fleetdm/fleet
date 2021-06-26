package fleet

import (
	"context"
	"time"
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

type Auth interface {
	UserID() string
	RequestID() string
}

type SessionService interface {
	// InitiateSSO is used to initiate an SSO session and returns a URL that
	// can be used in a redirect to the IDP.
	// Arguments: redirectURL is the URL of the protected resource that the user
	// was trying to access when they were promted to log in.
	InitiateSSO(ctx context.Context, redirectURL string) (string, error)
	// CallbackSSO handles the IDP response.  The original URL the viewer attempted
	// to access is returned from this function so we can redirect back to the front end and
	// load the page the viewer originally attempted to access when prompted for login.
	CallbackSSO(ctx context.Context, auth Auth) (*SSOSession, error)
	// SSOSettings returns non sensitive single sign on information used before
	// authentication
	SSOSettings(ctx context.Context) (*SSOSettings, error)
	Login(ctx context.Context, email, password string) (user *User, sessionKey string, err error)
	Logout(ctx context.Context) (err error)
	DestroySession(ctx context.Context) (err error)
	GetInfoAboutSessionsForUser(ctx context.Context, id uint) (sessions []*Session, err error)
	DeleteSessionsForUser(ctx context.Context, id uint) (err error)
	GetInfoAboutSession(ctx context.Context, id uint) (session *Session, err error)
	GetSessionByKey(ctx context.Context, key string) (session *Session, err error)
	DeleteSession(ctx context.Context, id uint) (err error)
}

type SSOSession struct {
	Token       string
	RedirectURL string
}

// SSOSettings SSO information used prior to authentication.
type SSOSettings struct {
	// IDPName is a human readable name for the IDP
	IDPName string `json:"idp_name"`
	// IDPImageURL https link to a logo image for the IDP.
	IDPImageURL string `json:"idp_image_url"`
	// SSOEnabled true if single sign on is enabled.
	SSOEnabled bool `json:"sso_enabled"`
}

// Session is the model object which represents what an active session is
type Session struct {
	CreateTimestamp
	ID         uint
	AccessedAt time.Time `db:"accessed_at"`
	UserID     uint      `json:"user_id" db:"user_id"`
	Key        string
}

func (s Session) AuthzType() string {
	return "session"
}
