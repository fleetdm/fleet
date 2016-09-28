package kolide

import (
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
)

// UserStore contains methods for managing users in a datastore
type UserStore interface {
	NewUser(user *User) (*User, error)
	User(username string) (*User, error)
	Users() ([]*User, error)
	UserByEmail(email string) (*User, error)
	UserByID(id uint) (*User, error)
	SaveUser(user *User) error
}

// UserService contains methods for managing a Kolide User
type UserService interface {
	// NewUser creates a new User from a request Payload
	NewUser(ctx context.Context, p UserPayload) (user *User, err error)

	// User returns a valid User given a User ID
	User(ctx context.Context, id uint) (user *User, err error)

	// AuthenticatedUser returns the current user
	// from the viewer context
	AuthenticatedUser(ctx context.Context) (user *User, err error)

	// Users returns all users
	Users(ctx context.Context) (users []*User, err error)

	// RequestPasswordReset generates a password reset request for
	// a user. The request results in a token emailed to the user.
	// If the person making the request is an admin the AdminForcedPasswordReset
	// parameter is enabled instead of sending an email with a password reset token
	RequestPasswordReset(ctx context.Context, email string) (err error)

	// ResetPassword validate a password reset token and updates
	// a user's password
	ResetPassword(ctx context.Context, token, password string) (err error)

	// ModifyUser updates a user's parameters given a UserPayload
	ModifyUser(ctx context.Context, userID uint, p UserPayload) (user *User, err error)
}

// User is the model struct which represents a kolide user
type User struct {
	ID                       uint `gorm:"primary_key"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
	Username                 string `gorm:"not null;unique_index:idx_user_unique_username"`
	Password                 []byte `gorm:"not null"`
	Salt                     string `gorm:"not null"`
	Name                     string
	Email                    string `gorm:"not null;unique_index:idx_user_unique_email"`
	Admin                    bool   `gorm:"not null"`
	Enabled                  bool   `gorm:"not null"`
	AdminForcedPasswordReset bool
	GravatarURL              string
	Position                 string // job role
}

// UserPayload is used to modify an existing user
type UserPayload struct {
	Username                 *string `json:"username"`
	Name                     *string `json:"name"`
	Email                    *string `json:"email"`
	Admin                    *bool   `json:"admin"`
	Enabled                  *bool   `json:"enabled"`
	AdminForcedPasswordReset *bool   `json:"force_password_reset"`
	Password                 *string `json:"password"`
	GravatarURL              *string `json:"gravatar_url"`
	Position                 *string `json:"position"`
}

// ValidatePassword accepts a potential password for a given user and attempts
// to validate it against the hash stored in the database after joining the
// supplied password with the stored password salt
func (u *User) ValidatePassword(password string) error {
	saltAndPass := []byte(fmt.Sprintf("%s%s", password, u.Salt))
	return bcrypt.CompareHashAndPassword(u.Password, saltAndPass)
}
