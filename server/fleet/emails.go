package fleet

import (
	"time"
)

// Mailer is an email campaign
// Types which implement the Campaign interface
// can be marshalled into an email body
type Mailer interface {
	Message() ([]byte, error)
}

type Email struct {
	Subject string
	To      []string
	Config  *AppConfig
	Mailer  Mailer
}

type MailService interface {
	SendEmail(e Email) error
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
