package fleet

import (
	"context"

	"gopkg.in/guregu/null.v3"
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

	// InviteByToken retrieves and invite using the token string.
	InviteByToken(token string) (*Invite, error)

	// DeleteInvite deletes an invitation.
	DeleteInvite(id uint) error
}

// InviteService contains methods for a service which deals with
// user invites.
type InviteService interface {
	// InviteNewUser creates an invite for a new user to join Fleet.
	InviteNewUser(ctx context.Context, payload InvitePayload) (invite *Invite, err error)

	// DeleteInvite removes an invite.
	DeleteInvite(ctx context.Context, id uint) (err error)

	// Invites returns a list of all invites.
	ListInvites(ctx context.Context, opt ListOptions) (invites []*Invite, err error)

	// VerifyInvite verifies that an invite exists and that it matches the
	// invite token.
	VerifyInvite(ctx context.Context, token string) (invite *Invite, err error)
}

// InvitePayload contains fields required to create a new user invite.
type InvitePayload struct {
	Email      *string
	Name       *string
	Position   *string
	SSOEnabled *bool       `json:"sso_enabled"`
	GlobalRole null.String `json:"global_role"`
	Teams      []UserTeam  `json:"teams"`
}

// Invite represents an invitation for a user to join Fleet.
type Invite struct {
	UpdateCreateTimestamps
	ID         uint        `json:"id"`
	InvitedBy  uint        `json:"invited_by" db:"invited_by"`
	Email      string      `json:"email"`
	Name       string      `json:"name"`
	Position   string      `json:"position,omitempty"`
	Token      string      `json:"-"`
	SSOEnabled bool        `json:"sso_enabled" db:"sso_enabled"`
	GlobalRole null.String `json:"global_role" db:"global_role"`
	Teams      []UserTeam  `json:"teams"`
}

func (i Invite) AuthzType() string {
	return "invite"
}
