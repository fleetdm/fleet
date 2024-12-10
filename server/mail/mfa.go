package mail

import (
	"bytes"
	"html/template"
	"time"

	"github.com/fleetdm/fleet/v4/server"
)

// MFAMailer is used to build an email template for the MFA email.
type MFAMailer struct {
	FullName    string
	Token       string
	BaseURL     template.URL
	AssetURL    template.URL
	CurrentYear int
}

func (i *MFAMailer) Message() ([]byte, error) {
	i.CurrentYear = time.Now().Year()
	t, err := server.GetTemplate("server/mail/templates/mfa.html", "email_template")
	if err != nil {
		return nil, err
	}

	var msg bytes.Buffer
	if err = t.Execute(&msg, i); err != nil {
		return nil, err
	}
	return msg.Bytes(), nil
}
