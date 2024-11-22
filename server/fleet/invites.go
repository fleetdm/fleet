package fleet

import (
	"gopkg.in/guregu/null.v3"
)

// InvitePayload contains fields required to create a new user invite or update an existing one.
type InvitePayload struct {
	Email      *string     `json:"email"`
	Name       *string     `json:"name"`
	Position   *string     `json:"position"`
	SSOEnabled *bool       `json:"sso_enabled"`
	GlobalRole null.String `json:"global_role"`
	Teams      []UserTeam  `json:"teams"`
}

// Invite represents an invitation for a user to join Fleet.
type Invite struct {
	UpdateCreateTimestamps
	ID         uint        `json:"id"`
	InvitedBy  string      `json:"invited_by" db:"invited_by"`
	Email      string      `json:"email" db:"email"`
	Name       string      `json:"name" db:"name"`
	Position   string      `json:"position,omitempty"`
	Token      string      `json:"-"`
	SSOEnabled bool        `json:"sso_enabled" db:"sso_enabled"`
	GlobalRole null.String `json:"global_role" db:"global_role"`
	Teams      []UserTeam  `json:"teams"`
}

func (i Invite) AuthzType() string {
	return "invite"
}
