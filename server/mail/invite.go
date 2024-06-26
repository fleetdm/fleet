package mail

import (
	"bytes"
	"html/template"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// InviteMailer is used to build an email template for the invite email.
type InviteMailer struct {
	*fleet.Invite
	BaseURL     template.URL
	AssetURL    template.URL
	InvitedBy   string
	OrgName     string
	CurrentYear int
}

func (i *InviteMailer) Message() ([]byte, error) {
	i.CurrentYear = time.Now().Year()
	t, err := server.GetTemplate("server/mail/templates/invite_token.html", "email_template")
	if err != nil {
		return nil, err
	}

	var msg bytes.Buffer
	if err = t.Execute(&msg, i); err != nil {
		return nil, err
	}
	return msg.Bytes(), nil
}
