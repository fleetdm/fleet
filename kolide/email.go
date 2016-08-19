package kolide

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jordan-wright/email"
	"github.com/kolide/kolide-ose/errors"
	"github.com/spf13/viper"
)

// CampaignStore manages email campaigns in the database
type EmailStore interface {
	CreatePassworResetRequest(userID uint, expires time.Time, token string) (*PasswordResetRequest, error)

	DeletePasswordResetRequest(req *PasswordResetRequest) error

	FindPassswordResetByID(id uint) (*PasswordResetRequest, error)

	FindPassswordResetByToken(token string) (*PasswordResetRequest, error)

	FindPassswordResetByTokenAndUserID(token string, id uint) (*PasswordResetRequest, error)
}

type EmailType int

const (
	PasswordResetEmail EmailType = iota
)

type PasswordResetRequestEmailParameters struct {
	Name  string
	Token string
}

const (
	NoReplyEmailAddress = "no-reply@kolide.co"
)

type SMTPConnectionPool interface {
	Send(e *email.Email, timeout time.Duration) error
	Close()
}

type mockSMTPConnectionPool struct {
	Emails []*email.Email
}

func newMockSMTPConnectionPool() *mockSMTPConnectionPool {
	return &mockSMTPConnectionPool{}
}

func (pool *mockSMTPConnectionPool) Send(e *email.Email, timeout time.Duration) error {
	pool.Emails = append(pool.Emails, e)
	return nil
}

func (pool *mockSMTPConnectionPool) Close() {}

func SendEmail(pool SMTPConnectionPool, to, subject string, html, text []byte) error {
	e := email.Email{
		From:    fmt.Sprintf("Kolide <%s>", NoReplyEmailAddress),
		To:      []string{to},
		Subject: subject,
		HTML:    html,
		Text:    text,
	}

	err := pool.Send(&e, time.Second*10)
	if err != nil {
		return errors.NewFromError(err, http.StatusInternalServerError, "Email error")
	}

	return nil
}

func GetEmailBody(t EmailType, params interface{}) (html []byte, text []byte, err error) {
	switch t {
	case PasswordResetEmail:
		resetParams, ok := params.(*PasswordResetRequestEmailParameters)
		if !ok {
			err = errors.New("Couldn't get email body", "Parameters were of incorrect type")
			return
		}

		html = []byte(fmt.Sprintf(
			"Hi %s! <a href=\"%s/password/reset?token=%s\">Reset your password!</a>",
			resetParams.Name,
			viper.GetString("app.web_address"),
			resetParams.Token,
		))
		text = []byte(fmt.Sprintf(
			"Hi %s! Reset your password: %s/password/reset?token=%s",
			resetParams.Name,
			viper.GetString("app.web_address"),
			resetParams.Token,
		))
	default:
		err = errors.New(
			"Couldn't get email body",
			fmt.Sprintf("Email type unknown: %d", t),
		)
	}
	return
}

func GetEmailSubject(t EmailType) (string, error) {
	switch t {
	case PasswordResetEmail:
		return "Your Kolide Password Reset Request", nil
	default:
		return "", errors.New(
			"Couldn't get email subject",
			fmt.Sprintf("Email type unknown: %d", t),
		)
	}
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

// NewPasswordResetRequest creates a password reset email campaign
func NewPasswordResetRequest(db EmailStore, userID uint, expires time.Time) (*PasswordResetRequest, error) {

	token, err := generateRandomText(viper.GetInt("smtp.token_key_size"))
	if err != nil {
		return nil, err
	}

	request, err := db.CreatePassworResetRequest(userID, expires, token)
	if err != nil {
		return nil, err
	}

	return request, nil
}
