package fleet

import (
	"time"
)

type Auth interface {
	UserID() string
	RequestID() string
}

type SSOSession struct {
	Token       string
	RedirectURL string
}

// SessionSSOSettings SSO information used prior to authentication.
type SessionSSOSettings struct {
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
