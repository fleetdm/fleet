package mail

import (
	"bytes"
	"html/template"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// InviteMailer is used to build an email template for the invite email.
type InviteMailer struct {
	*fleet.Invite
	BaseURL   template.URL
	AssetURL  template.URL
	InvitedBy string
	OrgName   string
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
