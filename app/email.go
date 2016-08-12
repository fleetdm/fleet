package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jordan-wright/email"
	"github.com/kolide/kolide-ose/errors"
	"github.com/spf13/viper"
)

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
