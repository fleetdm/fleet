package kolide

import (
	"bytes"
	"html/template"
	"time"
)

// PasswordResetStore manages password resets in the Datastore
type PasswordResetStore interface {
	NewPasswordResetRequest(req *PasswordResetRequest) (*PasswordResetRequest, error)
	SavePasswordResetRequest(req *PasswordResetRequest) error
	DeletePasswordResetRequest(req *PasswordResetRequest) error
	DeletePasswordResetRequestsForUser(userID uint) error
	FindPassswordResetByID(id uint) (*PasswordResetRequest, error)
	FindPassswordResetsByUserID(id uint) ([]*PasswordResetRequest, error)
	FindPassswordResetByToken(token string) (*PasswordResetRequest, error)
	FindPassswordResetByTokenAndUserID(token string, id uint) (*PasswordResetRequest, error)
}

// Mailer is an email campaign
// Types which implement the Campaign interface
// can be marshalled into an email body
type Mailer interface {
	Message() ([]byte, error)
}

type Email struct {
	To   []string
	From string
	Msg  Mailer
}

type MailService interface {
	SendEmail(e Email) error
}

// PasswordResetRequest represents a database table for
// Password Reset Requests
type PasswordResetRequest struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
	UserID    uint
	Token     string `gorm:"size:1024"`
}

const passwordResetTemplate = `
You requested a password reset,
Follow the link below to reset your password:
http://localhost:8080/login/reset?token={{.Token}}
`

func (r PasswordResetRequest) Message() ([]byte, error) {
	var msg bytes.Buffer
	var err error
	t := template.New(passwordResetTemplate)
	if t, err = t.Parse(passwordResetTemplate); err != nil {
		return nil, err
	}
	if err = t.Execute(&msg, r); err != nil {
		return nil, err
	}
	return msg.Bytes(), nil
}
