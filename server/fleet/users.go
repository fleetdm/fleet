package fleet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"time"
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
	SSOEnabled bool `json:"sso_enabled" db:"sso_enabled"`
	// MFAEnabled if true, the user (if non-SSO) must click a magic link via email to complete login
	MFAEnabled bool    `json:"mfa_enabled" db:"mfa_enabled"`
	GlobalRole *string `json:"global_role" db:"global_role"`
	APIOnly    bool    `json:"api_only" db:"api_only"`

	// Teams is the teams this user has roles in. For users with a global role, Teams is expected to be empty.
	Teams []UserTeam `json:"teams"`
}

type UserSettings struct {
	HiddenHostsTableColumns []string `json:"hidden_hosts_table_columns"`
}

// IsGlobalObserver returns true if user is either a Global Observer or a Global Observer+
func (u *User) IsGlobalObserver() bool {
	if u.GlobalRole == nil {
		return false
	}
	return *u.GlobalRole == RoleObserver || *u.GlobalRole == RoleObserverPlus
}

// TeamMembership returns a map whose keys are the TeamIDs of the teams for which pred evaluates to true
func (u *User) TeamMembership(pred func(UserTeam) bool) map[uint]bool {
	result := make(map[uint]bool)
	for _, t := range u.Teams {
		if pred(t) {
			result[t.ID] = true
		}
	}
	return result
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

func (u UserTeam) MarshalJSON() ([]byte, error) {
	x := struct {
		ID          uint      `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		TeamConfig
		UserCount int             `json:"user_count"`
		Users     []TeamUser      `json:"users,omitempty"`
		HostCount int             `json:"host_count"`
		Hosts     []HostResponse  `json:"hosts,omitempty"`
		Secrets   []*EnrollSecret `json:"secrets,omitempty"`
		Role      string          `json:"role"`
	}{
		ID:          u.ID,
		CreatedAt:   u.CreatedAt,
		Name:        u.Name,
		Description: u.Description,
		TeamConfig:  u.Config,
		UserCount:   u.UserCount,
		Users:       u.Users,
		HostCount:   u.HostCount,
		Hosts:       HostResponsesForHostsCheap(u.Hosts),
		Secrets:     u.Secrets,
		Role:        u.Role,
	}

	return json.Marshal(x)
}

func (u *UserTeam) UnmarshalJSON(b []byte) error {
	var x struct {
		ID          uint      `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		TeamConfig
		UserCount int             `json:"user_count"`
		Users     []TeamUser      `json:"users,omitempty"`
		HostCount int             `json:"host_count"`
		Hosts     []Host          `json:"hosts,omitempty"`
		Secrets   []*EnrollSecret `json:"secrets,omitempty"`
		Role      string          `json:"role"`
	}

	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}

	*u = UserTeam{
		Team: Team{
			ID:          x.ID,
			CreatedAt:   x.CreatedAt,
			Name:        x.Name,
			Description: x.Description,
			Config:      x.TeamConfig,
			UserCount:   x.UserCount,
			Users:       x.Users,
			HostCount:   x.HostCount,
			Hosts:       x.Hosts,
			Secrets:     x.Secrets,
		},
		Role: x.Role,
	}

	return nil
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
	MFAEnabled               *bool       `json:"mfa_enabled,omitempty"`
	SSOEnabled               *bool       `json:"sso_enabled,omitempty"`
	GlobalRole               *string     `json:"global_role,omitempty"`
	AdminForcedPasswordReset *bool       `json:"admin_forced_password_reset,omitempty"`
	APIOnly                  *bool       `json:"api_only,omitempty"`
	Teams                    *[]UserTeam `json:"teams,omitempty"`
	NewPassword              *string     `json:"new_password,omitempty"`
}

func (p *UserPayload) VerifyInviteCreate() error {
	invalid := &InvalidArgumentError{}
	p.verifyCreateShared(invalid)

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
	p.verifyCreateShared(invalid)

	if p.InviteToken != nil {
		invalid.Append("invite_token", "Invite token should not be specified with admin user creation")
	}

	if invalid.HasErrors() {
		return invalid
	}
	return nil
}

func (p *UserPayload) verifyCreateShared(invalid *InvalidArgumentError) {
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
		} else if err := ValidatePasswordRequirements(*p.Password); err != nil {
			invalid.Append("password", err.Error())
		}
	}

	if p.SSOEnabled != nil && *p.SSOEnabled {
		if p.Password != nil && len(*p.Password) > 0 {
			invalid.Append("password", "not allowed for SSO users")
		}
		if p.MFAEnabled != nil && *p.MFAEnabled {
			invalid.Append("mfa_enabled", "not applicable for SSO users")
		}
	}

	if p.Email == nil {
		invalid.Append("email", "Email missing required argument")
	} else if *p.Email == "" {
		invalid.Append("email", "Email cannot be empty")
	} else if err := ValidateEmail(*p.Email); err != nil {
		invalid.Append("email", err.Error())
	}
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
		// address a password is required.
		if ownUser && p.Password == nil {
			invalid.Append("password", "Password cannot be empty if email is changed")
		}
	}

	if p.SSOEnabled != nil && *p.SSOEnabled && p.NewPassword != nil && len(*p.NewPassword) > 0 {
		invalid.Append("new_password", "not allowed for SSO users")
	}
	if p.NewPassword != nil {
		// if the user is not an admin, or if an admin is changing their own password
		// a password is required.
		if ownUser && p.Password == nil {
			invalid.Append("password", "Old password cannot be empty")
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

	if (p.SSOInvite != nil && *p.SSOInvite) || (p.SSOEnabled != nil && *p.SSOEnabled) {
		user.SSOEnabled = true
		// SSO user requires a stand-in password to satisfy `NOT NULL` constraint
		err := user.SetFakePassword(keySize, cost)
		if err != nil {
			return nil, err
		}
	} else {
		err := user.SetPassword(*p.Password, keySize, cost)
		if err != nil {
			return nil, err
		}
		if p.MFAEnabled != nil {
			user.MFAEnabled = *p.MFAEnabled
		}
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
	if err := ValidatePasswordRequirements(plaintext); err != nil {
		return err
	}

	hashed, salt, err := saltAndHashPassword(keySize, plaintext, cost)
	if err != nil {
		return err
	}
	u.Password = hashed
	u.Salt = salt

	return nil
}

// ValidatePasswordRequirements checks the provided password against the following requirements:
// at least 12 character length
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

	if len(password) >= 12 &&
		number &&
		symbol {
		return nil
	}

	return errors.New("Password does not meet required criteria: Must include 12 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#).")
}

// ValidateEmail checks that the provided email address is valid, this function
// uses the stdlib func `mail.ParseAddress` underneath, which parses email
// adddresses using RFC5322, so it properly parses strings like "User
// <example.com>" thus we check if:
//
// 1. We're able to parse the address
// 2. The parsed address is equal to the provided address
func ValidateEmail(email string) error {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return err
	}

	if addr.Address != email {
		return errors.New("Email is invalid")
	}

	return nil
}

// SetFakePassword sets a stand-in password consisting of random text generated by filling in keySize bytes with
// random data and then base64 encoding those bytes.
//
// Usage should be limited to cases such as SSO users where a stand-in password is needed to satisfy `NOT NULL` constraints.
// There is no guarantee that the generated password will otherwise satisfy complexity, length or
// other requirements of standard password validation.
func (u *User) SetFakePassword(keySize, cost int) error {
	plaintext, err := server.GenerateRandomText(14)
	if err != nil {
		return err
	}

	hashed, salt, err := saltAndHashPassword(keySize, plaintext, cost)
	if err != nil {
		return err
	}
	u.Password = hashed
	u.Salt = salt

	return nil
}

func saltAndHashPassword(keySize int, plaintext string, cost int) (hashed []byte, salt string, err error) {
	salt, err = server.GenerateRandomText(keySize)
	if err != nil {
		return nil, "", err
	}

	salt = salt[:keySize]
	withSalt := []byte(fmt.Sprintf("%s%s", plaintext, salt))
	hashed, err = bcrypt.GenerateFromPassword(withSalt, cost)
	if err != nil {
		if errors.Is(err, bcrypt.ErrPasswordTooLong) {
			return nil, "", NewInvalidArgumentError("Could not create user. Password is over the 48 characters limit. If the password is under 48 characters, please check the auth_salt_key_size in your Fleet server config.", "password too long")
		}
		return nil, "", err
	}

	return hashed, salt, nil
}
