package fleet

import (
	"regexp"
	"time"
)

// Mailer is an email campaign
// Types which implement the Campaign interface
// can be marshalled into an email body
type Mailer interface {
	Message() ([]byte, error)
}

type Email struct {
	Subject      string
	To           []string
	ServerURL    string
	SMTPSettings SMTPSettings
	Mailer       Mailer
}

type MailService interface {
	SendEmail(e Email) error
	CanSendEmail(smtpSettings SMTPSettings) bool
}

// PasswordResetRequest represents a database table for
// Password Reset Requests
type PasswordResetRequest struct {
	UpdateCreateTimestamps
	ID        uint
	ExpiresAt time.Time `db:"expires_at"`
	UserID    uint      `db:"user_id"`
	Token     string
}

// very loosely checks that a string looks like an email:
// has no spaces, a single @ character, a part before the @,
// a part after the @, the part after has at least one dot
// with something after the dot. I don't think this is perfectly
// correct as the email format allows any chars including spaces
// when inside double quotes, but this is an edge case that is
// unlikely to matter much in practice. Another option that would
// definitely not cut out any valid address is to just check for
// the presence of @, which is arguably the most important check
// in this.
var rxLooseEmail = regexp.MustCompile(`^[^\s@]+@[^\s@\.]+\..+$`)

// IsLooseEmail loosely checks that the provided string looks like
// an email.
func IsLooseEmail(email string) bool {
	return rxLooseEmail.MatchString(email)
}
