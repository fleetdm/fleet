package contract

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	// If false/omitted, users that require email verification (Fleet MFA) to log in will fail to log in, rather than
	// sending an MFA email, since the MFA email will land the user in a browser and complete the login there, rather
	// than e.g. in the CLI that initiated the login. As with SSO, the expected behavior for users with MFA is to log
	// in with MFA, then grab an API token for use elsewhere.
	SupportsEmailVerification bool `json:"supports_email_verification"`
}
