package kolide

import (
	"bytes"
	"html/template"

	"golang.org/x/net/context"
)

// InviteStore contains the methods for
// managing user invites in a datastore.
type InviteStore interface {
	// NewInvite creates and stores a new invitation in a DB.
	NewInvite(i *Invite) (*Invite, error)

	// Invites lists all invites in the datastore.
	ListInvites(opt ListOptions) ([]*Invite, error)

	// Invite retrieves an invite by it's ID.
	Invite(id uint) (*Invite, error)

	// InviteByEmail retrieves an invite for a specific email address.
	InviteByEmail(email string) (*Invite, error)

	// SaveInvite saves an invitation in the datastore.
	SaveInvite(i *Invite) error

	// DeleteInvite deletes an invitation.
	DeleteInvite(i *Invite) error
}

// InviteService contains methods for a service which deals with
// user invites.
type InviteService interface {
	// InviteNewUser creates an invite for a new user to join Kolide.
	InviteNewUser(ctx context.Context, payload InvitePayload) (invite *Invite, err error)

	// DeleteInvite removes an invite.
	DeleteInvite(ctx context.Context, id uint) (err error)

	// Invites returns a list of all invites.
	ListInvites(ctx context.Context, opt ListOptions) (invites []*Invite, err error)

	// VerifyInvite verifies that an invite exists and that it matches the
	// invite token.
	VerifyInvite(ctx context.Context, email, token string) (err error)
}

// InvitePayload contains fields required to create a new user invite.
type InvitePayload struct {
	InvitedBy *uint `json:"invited_by"`
	Email     *string
	Admin     *bool
	Name      *string
	Position  *string
}

// Invite represents an invitation for a user to join Kolide.
type Invite struct {
	UpdateCreateTimestamps
	DeleteFields
	ID        uint   `json:"id" gorm:"primary_key"`
	InvitedBy uint   `json:"invited_by" gorm:"not null" db:"invited_by"`
	Email     string `json:"email" gorm:"not null;unique_index:idx_invite_unique_email"`
	Admin     bool   `json:"admin"`
	Name      string `json:"name"`
	Position  string `json:"position,omitempty"`
	Token     string `json:"-" gorm:"not null;unique_index:idx_invite_unique_key"`
}

// TODO: fixme
// this is not the right way to generate emails at all
const inviteEmailTempate = `
{{.InvitedBy}} invited you to join Kolide.,
http://localhost:8080/signup?token={{.Token}}
`

func (i Invite) Message() ([]byte, error) {
	var msg bytes.Buffer
	var err error
	t := template.New(inviteEmailTempate)
	if t, err = t.Parse(inviteEmailTempate); err != nil {
		return nil, err
	}
	if err = t.Execute(&msg, i); err != nil {
		return nil, err
	}
	return msg.Bytes(), nil
}
