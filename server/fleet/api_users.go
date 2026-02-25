package fleet

import "net/http"

type CreateUserRequest struct {
	UserPayload
}

type CreateUserResponse struct {
	User *User `json:"user,omitempty"`
	// Token is only returned when creating API-only (non-SSO) users.
	Token *string `json:"token,omitempty"`
	Err   error   `json:"error,omitempty"`
}

func (r CreateUserResponse) Error() error { return r.Err }

type ListUsersRequest struct {
	ListOptions UserListOptions `url:"user_options"`
}

type ListUsersResponse struct {
	Users []User `json:"users"`
	Err   error  `json:"error,omitempty"`
}

func (r ListUsersResponse) Error() error { return r.Err }

type GetMeRequest struct {
	IncludeUISettings bool `query:"include_ui_settings,optional"`
}

type GetUserRequest struct {
	ID                uint `url:"id"`
	IncludeUISettings bool `query:"include_ui_settings,optional"`
}

type GetUserResponse struct {
	User           *User          `json:"user,omitempty"`
	AvailableTeams []*TeamSummary `json:"available_teams" renameto:"available_fleets"`
	Settings       *UserSettings  `json:"settings,omitempty"`
	Err            error          `json:"error,omitempty"`
}

func (r GetUserResponse) Error() error { return r.Err }

type ModifyUserRequest struct {
	ID uint `json:"-" url:"id"`
	UserPayload
}

type ModifyUserResponse struct {
	User *User `json:"user,omitempty"`
	Err  error `json:"error,omitempty"`
}

func (r ModifyUserResponse) Error() error { return r.Err }

type DeleteUserRequest struct {
	ID uint `url:"id"`
}

type DeleteUserResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteUserResponse) Error() error { return r.Err }

type RequirePasswordResetRequest struct {
	Require bool `json:"require"`
	ID      uint `json:"-" url:"id"`
}

type RequirePasswordResetResponse struct {
	User *User `json:"user,omitempty"`
	Err  error `json:"error,omitempty"`
}

func (r RequirePasswordResetResponse) Error() error { return r.Err }

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type ChangePasswordResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ChangePasswordResponse) Error() error { return r.Err }

type GetInfoAboutSessionsForUserRequest struct {
	ID uint `url:"id"`
}

type GetInfoAboutSessionsForUserResponse struct {
	Sessions []GetInfoAboutSessionResponse `json:"sessions"`
	Err      error                         `json:"error,omitempty"`
}

func (r GetInfoAboutSessionsForUserResponse) Error() error { return r.Err }

type DeleteSessionsForUserRequest struct {
	ID uint `url:"id"`
}

type DeleteSessionsForUserResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteSessionsForUserResponse) Error() error { return r.Err }

type ChangeEmailRequest struct {
	Token string `url:"token"`
}

type ChangeEmailResponse struct {
	NewEmail string `json:"new_email"`
	Err      error  `json:"error,omitempty"`
}

func (r ChangeEmailResponse) Error() error { return r.Err }

type PerformRequiredPasswordResetRequest struct {
	Password string `json:"new_password"`
	ID       uint   `json:"id"`
}

type PerformRequiredPasswordResetResponse struct {
	User *User `json:"user,omitempty"`
	Err  error `json:"error,omitempty"`
}

func (r PerformRequiredPasswordResetResponse) Error() error { return r.Err }

type ResetPasswordRequest struct {
	PasswordResetToken string `json:"password_reset_token"`
	NewPassword        string `json:"new_password"`
}

type ResetPasswordResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ResetPasswordResponse) Error() error { return r.Err }

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ForgotPasswordResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ForgotPasswordResponse) Error() error { return r.Err }

func (r ForgotPasswordResponse) Status() int { return http.StatusAccepted }
