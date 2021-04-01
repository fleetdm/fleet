package kolide

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/guregu/null.v3"
)

// UserStore contains methods for managing users in a datastore
type UserStore interface {
	NewUser(user *User) (*User, error)
	User(username string) (*User, error)
	ListUsers(opt ListOptions) ([]*User, error)
	UserByEmail(email string) (*User, error)
	UserByID(id uint) (*User, error)
	SaveUser(user *User) error
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
	CreateUserWithInvite(ctx context.Context, p UserPayload) (user *User, err error)

	// CreateUser allows an admin to create a new user without first creating
	// and validating invite tokens.
	CreateUser(ctx context.Context, p UserPayload) (user *User, err error)

	// User returns a valid User given a User ID.
	User(ctx context.Context, id uint) (user *User, err error)

	// AuthenticatedUser returns the current user from the viewer context.
	AuthenticatedUser(ctx context.Context) (user *User, err error)

	// ListUsers returns all users.
	ListUsers(ctx context.Context, opt ListOptions) (users []*User, err error)

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

	// ChangeUserAdmin is used to modify the admin state of the user identified by id.
	ChangeUserAdmin(ctx context.Context, id uint, isAdmin bool) (*User, error)

	// ChangeUserEnabled is used to enable/disable the user identified by id.
	ChangeUserEnabled(ctx context.Context, id uint, isEnabled bool) (*User, error)

	// ChangeUserEmail is used to confirm new email address and if confirmed,
	// write the new email address to user.
	ChangeUserEmail(ctx context.Context, token string) (string, error)
}

// User is the model struct which represents a kolide user
type User struct {
	UpdateCreateTimestamps
	ID                       uint   `json:"id"`
	Username                 string `json:"username"`
	Password                 []byte `json:"-"`
	Salt                     string `json:"-"`
	Name                     string `json:"name"`
	Email                    string `json:"email"`
	Admin                    bool   `json:"admin"`
	Enabled                  bool   `json:"enabled"`
	AdminForcedPasswordReset bool   `json:"force_password_reset" db:"admin_forced_password_reset"`
	GravatarURL              string `json:"gravatar_url" db:"gravatar_url"`
	Position                 string `json:"position,omitempty"` // job role
	// SSOEnabled if true, the user may only log in via SSO
	SSOEnabled bool        `json:"sso_enabled" db:"sso_enabled"`
	GlobalRole null.String `json:"global_role" db:"global_role"`

	// Teams is the teams this user has roles in.
	Teams []UserTeam `json:"teams"`
}

type UserTeam struct {
	// Team is the team object.
	Team
	// Role is the role the user has for the team.
	Role string `json:"role" db:"role"`
}

// UserPayload is used to modify an existing user
type UserPayload struct {
	Username                 *string     `json:"username,omitempty"`
	Name                     *string     `json:"name,omitempty"`
	Email                    *string     `json:"email,omitempty"`
	Admin                    *bool       `json:"admin,omitempty"`
	Enabled                  *bool       `json:"enabled,omitempty"`
	Password                 *string     `json:"password,omitempty"`
	GravatarURL              *string     `json:"gravatar_url,omitempty"`
	Position                 *string     `json:"position,omitempty"`
	InviteToken              *string     `json:"invite_token,omitempty"`
	SSOInvite                *bool       `json:"sso_invite,omitempty"`
	SSOEnabled               *bool       `json:"sso_enabled,omitempty"`
	GlobalRole               *string     `json:"global_role,omitempty"`
	AdminForcedPasswordReset *bool       `json:"admin_forced_password_reset,omitempty"`
	Teams                    *[]UserTeam `json:"teams,omitempty"`
}

// User creates a user from payload.
func (p UserPayload) User(keySize, cost int) (*User, error) {
	user := &User{
		Username: *p.Username,
		Email:    *p.Email,
		Admin:    falseIfNil(p.Admin),
		Enabled:  true,
		Teams:    []UserTeam{},
	}
	if err := user.SetPassword(*p.Password, keySize, cost); err != nil {
		return nil, err
	}

	// add optional fields
	if p.Name != nil {
		user.Name = *p.Name
	}
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
	if p.Teams != nil {
		user.Teams = *p.Teams
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
	salt, err := generateRandomText(keySize)
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

// generateRandomText return a string generated by filling in keySize bytes with
// random data and then base64 encoding those bytes
func generateRandomText(keySize int) (string, error) {
	key := make([]byte, keySize)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// helper to convert a bool pointer false
func falseIfNil(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
