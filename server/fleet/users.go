package fleet

import (
	"context"
	"fmt"
	"github.com/fleetdm/fleet/v4/server"
	"golang.org/x/crypto/bcrypt"
)

// UserStore contains methods for managing users in a datastore
type UserStore interface {
	NewUser(user *User) (*User, error)
	ListUsers(opt UserListOptions) ([]*User, error)
	UserByEmail(email string) (*User, error)
	UserByID(id uint) (*User, error)
	SaveUser(user *User) error
	SaveUsers(users []*User) error
	// DeleteUser permanently deletes the user identified by the provided ID.
	DeleteUser(id uint) error
	// PendingEmailChange creates a record with a pending email change for a user identified
	// by uid. The change record is keyed by a unique token. The token is emailed to the user
	// with a link that they can use to confirm the change.
	PendingEmailChange(userID uint, newEmail, token string) error
	// ConfirmPendingEmailChange will confirm new email address identified by token is valid.
	// The new email will be written to user record. userID is the ID of the
	// user whose e-mail is being changed.
	ConfirmPendingEmailChange(userID uint, token string) (string, error)
}

// UserService contains methods for managing a Fleet User.
type UserService interface {
	// CreateUserWithInvite creates a new User from a request payload when there is
	// already an existing invitation.
	CreateUserFromInvite(ctx context.Context, p UserPayload) (user *User, err error)

	// CreateUser allows an admin to create a new user without first creating
	// and validating invite tokens.
	CreateUser(ctx context.Context, p UserPayload) (user *User, err error)

	// CreateInitialUser creates the first user, skipping authorization checks.
	// If a user already exists this method should fail.
	CreateInitialUser(ctx context.Context, p UserPayload) (user *User, err error)

	// User returns a valid User given a User ID.
	User(ctx context.Context, id uint) (user *User, err error)

	// UserUnauthorized returns a valid User given a User ID, *skipping authorization checks*
	//
	// This method should only be used in middleware where there is not yet a viewer context and we need to load up a user to create that context.
	UserUnauthorized(ctx context.Context, id uint) (user *User, err error)

	// AuthenticatedUser returns the current user from the viewer context.
	AuthenticatedUser(ctx context.Context) (user *User, err error)

	// ListUsers returns all users.
	ListUsers(ctx context.Context, opt UserListOptions) (users []*User, err error)

	// ChangePassword validates the existing password, and sets the new
	// password. User is retrieved from the viewer context.
	ChangePassword(ctx context.Context, oldPass, newPass string) error

	// RequestPasswordReset generates a password reset request for the user
	// specified by email. The request results in a token emailed to the
	// user.
	RequestPasswordReset(ctx context.Context, email string) (err error)

	// RequirePasswordReset requires a password reset for the user
	// specified by ID (if require is true). It deletes all of the user's
	// sessions, and requires that their password be reset upon the next
	// login. Setting require to false will take a user out of this state.
	// The updated user is returned.
	RequirePasswordReset(ctx context.Context, uid uint, require bool) (*User, error)

	// PerformRequiredPasswordReset resets a password for a user that is in
	// the required reset state. It must be called with the logged in
	// viewer context of that user.
	PerformRequiredPasswordReset(ctx context.Context, password string) (*User, error)

	// ResetPassword validates the provided password reset token and
	// updates the user's password.
	ResetPassword(ctx context.Context, token, password string) (err error)

	// ModifyUser updates a user's parameters given a UserPayload.
	ModifyUser(ctx context.Context, userID uint, p UserPayload) (user *User, err error)

	// DeleteUser permanently deletes the user identified by the provided ID.
	DeleteUser(ctx context.Context, id uint) error

	// ChangeUserEmail is used to confirm new email address and if confirmed,
	// write the new email address to user.
	ChangeUserEmail(ctx context.Context, token string) (string, error)
}

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

	// Teams is the teams this user has roles in.
	Teams []UserTeam `json:"teams"`
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

// helper to convert a bool pointer false
func falseIfNil(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
