package fleet

import (
	"errors"
	"fmt"
	"unicode"

	"github.com/fleetdm/fleet/v4/server"
	"golang.org/x/crypto/bcrypt"
)

// User is the model struct that represents a Fleet user.
type User struct {
	UpdateCreateTimestamps
	ID                       uint   `json:"id"`
	Password                 []byte `json:"-"`
	Salt                     string `json:"-"`
	Name                     string `json:"name"`
	Email                    string `json:"email"`
	AdminForcedPasswordReset bool   `json:"force_password_reset" db:"admin_forced_password_reset"`
	GravatarURL              string `json:"gravatar_url" db:"gravatar_url"`
	Position                 string `json:"position,omitempty"` // job role
	// SSOEnabled if true, the user may only log in via SSO
	SSOEnabled bool    `json:"sso_enabled" db:"sso_enabled"`
	GlobalRole *string `json:"global_role" db:"global_role"`
	APIOnly    bool    `json:"api_only" db:"api_only"`

	// Teams is the teams this user has roles in. For users with a global role, Teams is expected to be empty.
	Teams []UserTeam `json:"teams"`
}

func (u *User) IsAdminForcedPasswordReset() bool {
	if u.SSOEnabled {
		return false
	}
	return u.AdminForcedPasswordReset
}

func (u *User) AuthzType() string {
	return "user"
}

type UserTeam struct {
	// Team is the team object.
	Team
	// Role is the role the user has for the team.
	Role string `json:"role" db:"role"`
}

// UserListOptions is additional options that can be set for listing users.
type UserListOptions struct {
	ListOptions

	// TeamID, if set, indicates to only return members of the identified team.
	TeamID uint
}

// UserPayload is used to modify an existing user
type UserPayload struct {
	Name                     *string     `json:"name,omitempty"`
	Email                    *string     `json:"email,omitempty"`
	Password                 *string     `json:"password,omitempty"`
	GravatarURL              *string     `json:"gravatar_url,omitempty"`
	Position                 *string     `json:"position,omitempty"`
	InviteToken              *string     `json:"invite_token,omitempty"`
	SSOInvite                *bool       `json:"sso_invite,omitempty"`
	SSOEnabled               *bool       `json:"sso_enabled,omitempty"`
	GlobalRole               *string     `json:"global_role,omitempty"`
	AdminForcedPasswordReset *bool       `json:"admin_forced_password_reset,omitempty"`
	APIOnly                  *bool       `json:"api_only,omitempty"`
	Teams                    *[]UserTeam `json:"teams,omitempty"`
}

func (p *UserPayload) VerifyInviteCreate() error {
	invalid := &InvalidArgumentError{}
	if p.Name == nil {
		invalid.Append("name", "Full name missing required argument")
	} else if *p.Name == "" {
		invalid.Append("name", "Full name cannot be empty")
	}

	// we don't need a password for single sign on
	if p.SSOInvite == nil || !*p.SSOInvite {
		if p.Password == nil {
			invalid.Append("password", "Password missing required argument")
		} else if *p.Password == "" {
			invalid.Append("password", "Password cannot be empty")
		} else if err := ValidatePasswordRequirements(*p.Password); err != nil {
			invalid.Append("password", err.Error())
		}
	}

	if p.Email == nil {
		invalid.Append("email", "Email missing required argument")
	} else if *p.Email == "" {
		invalid.Append("email", "Email cannot be empty")
	}

	if p.InviteToken == nil {
		invalid.Append("invite_token", "Invite token missing required argument")
	} else if *p.InviteToken == "" {
		invalid.Append("invite_token", "Invite token cannot be empty")
	}

	if invalid.HasErrors() {
		return invalid
	}
	return nil
}

func (p *UserPayload) VerifyAdminCreate() error {
	invalid := &InvalidArgumentError{}
	if p.Name == nil {
		invalid.Append("name", "Full name missing required argument")
	} else if *p.Name == "" {
		invalid.Append("name", "Full name cannot be empty")
	}

	// we don't need a password for single sign on
	if (p.SSOInvite == nil || !*p.SSOInvite) && (p.SSOEnabled == nil || !*p.SSOEnabled) {
		if p.Password == nil {
			invalid.Append("password", "Password missing required argument")
		} else if *p.Password == "" {
			invalid.Append("password", "Password cannot be empty")
		}
		// Skip password validation in the case of admin created users
	}

	if p.SSOEnabled != nil && *p.SSOEnabled && p.Password != nil && len(*p.Password) > 0 {
		invalid.Append("password", "not allowed for SSO users")
	}

	if p.Email == nil {
		invalid.Append("email", "Email missing required argument")
	} else if *p.Email == "" {
		invalid.Append("email", "Email cannot be empty")
	}

	if p.InviteToken != nil {
		invalid.Append("invite_token", "Invite token should not be specified with admin user creation")
	}

	if invalid.HasErrors() {
		return invalid
	}
	return nil
}

func (p *UserPayload) VerifyModify(ownUser bool) error {
	invalid := &InvalidArgumentError{}
	if p.Name != nil && *p.Name == "" {
		invalid.Append("name", "Full name cannot be empty")
	}

	if p.Email != nil {
		if *p.Email == "" {
			invalid.Append("email", "Email cannot be empty")
		}
		// if the user is not an admin, or if an admin is changing their own email
		// address a password is required,
		if ownUser && p.Password == nil {
			invalid.Append("password", "Password cannot be empty if email is changed")
		}
	}

	if invalid.HasErrors() {
		return invalid
	}
	return nil
}

// User creates a user from payload.
func (p UserPayload) User(keySize, cost int) (*User, error) {
	user := &User{
		Name:  *p.Name,
		Email: *p.Email,
		Teams: []UserTeam{},
	}
	if err := user.SetPassword(*p.Password, keySize, cost); err != nil {
		return nil, err
	}

	// add optional fields
	if p.GravatarURL != nil {
		user.GravatarURL = *p.GravatarURL
	}
	if p.Position != nil {
		user.Position = *p.Position
	}
	if p.SSOEnabled != nil {
		user.SSOEnabled = *p.SSOEnabled
	}
	if p.AdminForcedPasswordReset != nil {
		user.AdminForcedPasswordReset = *p.AdminForcedPasswordReset
	}
	if p.APIOnly != nil {
		user.APIOnly = *p.APIOnly
	}
	if p.Teams != nil {
		user.Teams = *p.Teams
	}
	if p.GlobalRole != nil {
		user.GlobalRole = p.GlobalRole
	}

	return user, nil
}

// ValidatePassword accepts a potential password for a given user and attempts
// to validate it against the hash stored in the database after joining the
// supplied password with the stored password salt
func (u *User) ValidatePassword(password string) error {
	saltAndPass := []byte(fmt.Sprintf("%s%s", password, u.Salt))
	return bcrypt.CompareHashAndPassword(u.Password, saltAndPass)
}

func (u *User) SetPassword(plaintext string, keySize, cost int) error {
	salt, err := server.GenerateRandomText(keySize)
	if err != nil {
		return err
	}

	withSalt := []byte(fmt.Sprintf("%s%s", plaintext, salt))
	hashed, err := bcrypt.GenerateFromPassword(withSalt, cost)
	if err != nil {
		return err
	}

	u.Salt = salt
	u.Password = hashed
	return nil
}

// Requirements for user password:
// at least 7 character length
// at least 1 symbol
// at least 1 number
func ValidatePasswordRequirements(password string) error {
	var (
		number bool
		symbol bool
	)

	for _, s := range password {
		switch {
		case unicode.IsNumber(s):
			number = true
		case unicode.IsPunct(s) || unicode.IsSymbol(s):
			symbol = true
		}
	}

	if len(password) >= 7 &&
		number &&
		symbol {
		return nil
	}

	return errors.New("Password does not meet validation requirements")
}
