package mail

import (
	"bytes"
	"html/template"

	"github.com/kolide/fleet/server/kolide"
)

// InviteMailer is used to build an email template for the invite email.
type InviteMailer struct {
	*kolide.Invite
	BaseURL           template.URL
	AssetURL          template.URL
	InvitedByUsername string
	OrgName           string
}

func (i *InviteMailer) Message() ([]byte, error) {
	t, err := getTemplate("server/mail/templates/invite_token.html")
	if err != nil {
		return nil, err
	}

	var msg bytes.Buffer
	if err = t.Execute(&msg, i); err != nil {
		return nil, err
	}
	return msg.Bytes(), nil
}
