package mail

import (
	"bytes"
	"html/template"
	"time"

	"github.com/fleetdm/fleet/v4/server"
)

type ChangeEmailMailer struct {
	BaseURL     template.URL
	AssetURL    template.URL
	Token       string
	CurrentYear int
}

func (cem *ChangeEmailMailer) Message() ([]byte, error) {
	cem.CurrentYear = time.Now().Year()
	t, err := server.GetTemplate("server/mail/templates/change_email_confirmation.html", "email_template")
	if err != nil {
		return nil, err
	}
	var msg bytes.Buffer
	err = t.Execute(&msg, cem)
	if err != nil {
		return nil, err
	}
	return msg.Bytes(), nil
}

type PasswordResetMailer struct {
	// Base URL to use for Fleet endpoints
	BaseURL template.URL
	// URL for loading image assets
	AssetURL template.URL
	// Token password reset token
	Token string
	// Current year for copyright year
	CurrentYear int
}

func (r PasswordResetMailer) Message() ([]byte, error) {
	r.CurrentYear = time.Now().Year() // nolint:staticcheck // SA4005 false positive for Go templates
	t, err := server.GetTemplate("server/mail/templates/password_reset.html", "email_template")
	if err != nil {
		return nil, err
	}

	var msg bytes.Buffer
	if err = t.Execute(&msg, r); err != nil {
		return nil, err
	}
	return msg.Bytes(), nil
}
