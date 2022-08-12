package fleet

import (
	"time"
)

// Auth contains methods to fetch information from a valid SSO auth response
type Auth interface {
	// UserID returns the Subject Name Identifier associated with the request,
	// this can be an email address, an entity identifier, or any other valid
	// Name Identifier as described in the spec:
	// http://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf
	//
	// Fleet requires users to configure this value to be the email of the Subject
	UserID() string
	// UserDisplayName finds a display name in the SSO response Attributes, there
	// isn't a defined spec for this, so the return value is in a best-effort
	// basis
	UserDisplayName() string
	// RequestID returns the request id associated with this SSO session
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
	APIOnly    *bool `json:"-" db:"api_only"`
}

func (s Session) AuthzType() string {
	return "session"
}
