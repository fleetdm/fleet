package service

import (
	"context"
	"encoding/base64"
	"errors"
	"html/template"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mail"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

////////////////////////////////////////////////////////////////////////////////
// Create User
////////////////////////////////////////////////////////////////////////////////

type createUserRequest struct {
	fleet.UserPayload
}

type createUserResponse struct {
	User *fleet.User `json:"user,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r createUserResponse) error() error { return r.Err }

func createUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*createUserRequest)
	user, err := svc.CreateUser(ctx, req.UserPayload)
	if err != nil {
		return createUserResponse{Err: err}, nil
	}
	return createUserResponse{User: user}, nil
}

func (svc *Service) CreateUser(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	var teams []fleet.UserTeam
	if p.Teams != nil {
		teams = *p.Teams
	}
	if err := svc.authz.Authorize(ctx, &fleet.User{Teams: teams}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if invite, err := svc.ds.InviteByEmail(ctx, *p.Email); err == nil && invite != nil {
		return nil, ctxerr.Errorf(ctx, "%s already invited", *p.Email)
	}

	if p.AdminForcedPasswordReset == nil {
		// By default, force password reset for users created this way.
		p.AdminForcedPasswordReset = ptr.Bool(true)
	}

	return svc.newUser(ctx, p)
}

////////////////////////////////////////////////////////////////////////////////
// List Users
////////////////////////////////////////////////////////////////////////////////

type listUsersRequest struct {
	ListOptions fleet.UserListOptions `url:"user_options"`
}

type listUsersResponse struct {
	Users []fleet.User `json:"users"`
	Err   error        `json:"error,omitempty"`
}

func (r listUsersResponse) error() error { return r.Err }

func listUsersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listUsersRequest)
	users, err := svc.ListUsers(ctx, req.ListOptions)
	if err != nil {
		return listUsersResponse{Err: err}, nil
	}

	resp := listUsersResponse{Users: []fleet.User{}}
	for _, user := range users {
		resp.Users = append(resp.Users, *user)
	}
	return resp, nil
}

func (svc *Service) ListUsers(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
	if err := svc.authz.Authorize(ctx, &fleet.User{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListUsers(ctx, opt)
}

////////////////////////////////////////////////////////////////////////////////
// Me (get own current user)
////////////////////////////////////////////////////////////////////////////////

func meEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	user, err := svc.AuthenticatedUser(ctx)
	if err != nil {
		return getUserResponse{Err: err}, nil
	}
	availableTeams, err := svc.ListAvailableTeamsForUser(ctx, user)
	if err != nil {
		if errors.Is(err, fleet.ErrMissingLicense) {
			availableTeams = []*fleet.TeamSummary{}
		} else {
			return getUserResponse{Err: err}, nil
		}
	}
	return getUserResponse{User: user, AvailableTeams: availableTeams}, nil
}

func (svc *Service) AuthenticatedUser(ctx context.Context) (*fleet.User, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	if err := svc.authz.Authorize(ctx, &fleet.User{ID: vc.UserID()}, fleet.ActionRead); err != nil {
		return nil, err
	}

	if !vc.IsLoggedIn() {
		return nil, fleet.NewPermissionError("not logged in")
	}
	return vc.User, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get User
////////////////////////////////////////////////////////////////////////////////

type getUserRequest struct {
	ID uint `url:"id"`
}

type getUserResponse struct {
	User           *fleet.User          `json:"user,omitempty"`
	AvailableTeams []*fleet.TeamSummary `json:"available_teams"`
	Err            error                `json:"error,omitempty"`
}

func (r getUserResponse) error() error { return r.Err }

func getUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getUserRequest)
	user, err := svc.User(ctx, req.ID)
	if err != nil {
		return getUserResponse{Err: err}, nil
	}
	availableTeams, err := svc.ListAvailableTeamsForUser(ctx, user)
	if err != nil {
		if errors.Is(err, fleet.ErrMissingLicense) {
			availableTeams = []*fleet.TeamSummary{}
		} else {
			return getUserResponse{Err: err}, nil
		}
	}
	return getUserResponse{User: user, AvailableTeams: availableTeams}, nil
}

func (svc *Service) User(ctx context.Context, id uint) (*fleet.User, error) {
	if err := svc.authz.Authorize(ctx, &fleet.User{ID: id}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.UserByID(ctx, id)
}

////////////////////////////////////////////////////////////////////////////////
// Modify User
////////////////////////////////////////////////////////////////////////////////

type modifyUserRequest struct {
	ID uint `json:"-" url:"id"`
	fleet.UserPayload
}

type modifyUserResponse struct {
	User *fleet.User `json:"user,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r modifyUserResponse) error() error { return r.Err }

func modifyUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*modifyUserRequest)
	user, err := svc.ModifyUser(ctx, req.ID, req.UserPayload)
	if err != nil {
		return modifyUserResponse{Err: err}, nil
	}

	return modifyUserResponse{User: user}, nil
}

func (svc *Service) ModifyUser(ctx context.Context, userID uint, p fleet.UserPayload) (*fleet.User, error) {
	if err := svc.authz.Authorize(ctx, &fleet.User{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	user, err := svc.User(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := svc.authz.Authorize(ctx, user, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if p.GlobalRole != nil || p.Teams != nil {
		if err := svc.authz.Authorize(ctx, user, fleet.ActionWriteRole); err != nil {
			return nil, err
		}
	}
	if p.Name != nil {
		user.Name = *p.Name
	}

	if p.Email != nil && *p.Email != user.Email {
		err = svc.modifyEmailAddress(ctx, user, *p.Email, p.Password)
		if err != nil {
			return nil, err
		}
	}

	if p.Position != nil {
		user.Position = *p.Position
	}

	if p.GravatarURL != nil {
		user.GravatarURL = *p.GravatarURL
	}

	if p.SSOEnabled != nil {
		user.SSOEnabled = *p.SSOEnabled
	}

	currentUser := authz.UserFromContext(ctx)

	if p.GlobalRole != nil && *p.GlobalRole != "" {
		if currentUser.GlobalRole == nil {
			return nil, ctxerr.New(ctx, "Cannot edit global role as a team member")
		}

		if p.Teams != nil && len(*p.Teams) > 0 {
			return nil, fleet.NewInvalidArgumentError("teams", "may not be specified with global_role")
		}
		user.GlobalRole = p.GlobalRole
		user.Teams = []fleet.UserTeam{}
	} else if p.Teams != nil {
		if !isAdminOfTheModifiedTeams(currentUser, user.Teams, *p.Teams) {
			return nil, ctxerr.New(ctx, "Cannot modify teams in that way")
		}
		user.Teams = *p.Teams
		user.GlobalRole = nil
	}

	err = svc.saveUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

////////////////////////////////////////////////////////////////////////////////
// Delete User
////////////////////////////////////////////////////////////////////////////////

type deleteUserRequest struct {
	ID uint `url:"id"`
}

type deleteUserResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteUserResponse) error() error { return r.Err }

func deleteUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deleteUserRequest)
	err := svc.DeleteUser(ctx, req.ID)
	if err != nil {
		return deleteUserResponse{Err: err}, nil
	}
	return deleteUserResponse{}, nil
}

func (svc *Service) DeleteUser(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.User{ID: id}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DeleteUser(ctx, id)
}

////////////////////////////////////////////////////////////////////////////////
// Require Password Reset
////////////////////////////////////////////////////////////////////////////////

type requirePasswordResetRequest struct {
	Require bool `json:"require"`
	ID      uint `json:"-" url:"id"`
}

type requirePasswordResetResponse struct {
	User *fleet.User `json:"user,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r requirePasswordResetResponse) error() error { return r.Err }

func requirePasswordResetEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*requirePasswordResetRequest)
	user, err := svc.RequirePasswordReset(ctx, req.ID, req.Require)
	if err != nil {
		return requirePasswordResetResponse{Err: err}, nil
	}
	return requirePasswordResetResponse{User: user}, nil
}

func (svc *Service) RequirePasswordReset(ctx context.Context, uid uint, require bool) (*fleet.User, error) {
	if err := svc.authz.Authorize(ctx, &fleet.User{ID: uid}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	user, err := svc.ds.UserByID(ctx, uid)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading user by ID")
	}
	if user.SSOEnabled {
		return nil, ctxerr.New(ctx, "password reset for single sign on user not allowed")
	}
	// Require reset on next login
	user.AdminForcedPasswordReset = require
	if err := svc.saveUser(ctx, user); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "saving user")
	}

	if require {
		// Clear all of the existing sessions
		if err := svc.DeleteSessionsForUser(ctx, user.ID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "deleting user sessions")
		}
	}

	return user, nil
}

////////////////////////////////////////////////////////////////////////////////
// Change Password
////////////////////////////////////////////////////////////////////////////////

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type changePasswordResponse struct {
	Err error `json:"error,omitempty"`
}

func (r changePasswordResponse) error() error { return r.Err }

func changePasswordEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*changePasswordRequest)
	err := svc.ChangePassword(ctx, req.OldPassword, req.NewPassword)
	return changePasswordResponse{Err: err}, nil
}

func (svc *Service) ChangePassword(ctx context.Context, oldPass, newPass string) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	if err := svc.authz.Authorize(ctx, vc.User, fleet.ActionWrite); err != nil {
		return err
	}

	if vc.User.SSOEnabled {
		return ctxerr.New(ctx, "change password for single sign on user not allowed")
	}
	if err := vc.User.ValidatePassword(newPass); err == nil {
		return fleet.NewInvalidArgumentError("new_password", "cannot reuse old password")
	}

	if err := vc.User.ValidatePassword(oldPass); err != nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("old_password", "old password does not match"))
	}

	if err := svc.setNewPassword(ctx, vc.User, newPass); err != nil {
		return ctxerr.Wrap(ctx, err, "setting new password")
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Info About Sessions For User
////////////////////////////////////////////////////////////////////////////////

type getInfoAboutSessionsForUserRequest struct {
	ID uint `url:"id"`
}

type getInfoAboutSessionsForUserResponse struct {
	Sessions []getInfoAboutSessionResponse `json:"sessions"`
	Err      error                         `json:"error,omitempty"`
}

func (r getInfoAboutSessionsForUserResponse) error() error { return r.Err }

func getInfoAboutSessionsForUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getInfoAboutSessionsForUserRequest)
	sessions, err := svc.GetInfoAboutSessionsForUser(ctx, req.ID)
	if err != nil {
		return getInfoAboutSessionsForUserResponse{Err: err}, nil
	}
	var resp getInfoAboutSessionsForUserResponse
	for _, session := range sessions {
		resp.Sessions = append(resp.Sessions, getInfoAboutSessionResponse{
			SessionID: session.ID,
			UserID:    session.UserID,
			CreatedAt: session.CreatedAt,
		})
	}
	return resp, nil
}

func (svc *Service) GetInfoAboutSessionsForUser(ctx context.Context, id uint) ([]*fleet.Session, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Session{UserID: id}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	var validatedSessions []*fleet.Session

	sessions, err := svc.ds.ListSessionsForUser(ctx, id)
	if err != nil {
		return validatedSessions, err
	}

	for _, session := range sessions {
		if svc.validateSession(ctx, session) == nil {
			validatedSessions = append(validatedSessions, session)
		}
	}

	return validatedSessions, nil
}

////////////////////////////////////////////////////////////////////////////////
// Delete Sessions For User
////////////////////////////////////////////////////////////////////////////////

type deleteSessionsForUserRequest struct {
	ID uint `url:"id"`
}

type deleteSessionsForUserResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteSessionsForUserResponse) error() error { return r.Err }

func deleteSessionsForUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deleteSessionsForUserRequest)
	err := svc.DeleteSessionsForUser(ctx, req.ID)
	if err != nil {
		return deleteSessionsForUserResponse{Err: err}, nil
	}
	return deleteSessionsForUserResponse{}, nil
}

func (svc *Service) DeleteSessionsForUser(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Session{UserID: id}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DestroyAllSessionsForUser(ctx, id)
}

func isAdminOfTheModifiedTeams(currentUser *fleet.User, originalUserTeams, newUserTeams []fleet.UserTeam) bool {
	// If the user is of the right global role, then they can modify the teams
	if currentUser.GlobalRole != nil && (*currentUser.GlobalRole == fleet.RoleAdmin || *currentUser.GlobalRole == fleet.RoleMaintainer) {
		return true
	}

	// otherwise, gather the resulting teams
	resultingTeams := make(map[uint]string)
	for _, team := range newUserTeams {
		resultingTeams[team.ID] = team.Role
	}

	// and see which ones were removed or changed from the original
	teamsAffected := make(map[uint]struct{})
	for _, team := range originalUserTeams {
		if resultingTeams[team.ID] != team.Role {
			teamsAffected[team.ID] = struct{}{}
		}
	}

	// then gather the teams the current user is admin for
	currentUserTeamAdmin := make(map[uint]struct{})
	for _, team := range currentUser.Teams {
		if team.Role == fleet.RoleAdmin {
			currentUserTeamAdmin[team.ID] = struct{}{}
		}
	}

	// and let's check that the teams that were either removed or changed are also teams this user is an admin of
	for teamID := range teamsAffected {
		if _, ok := currentUserTeamAdmin[teamID]; !ok {
			return false
		}
	}

	return true
}

func (svc *Service) modifyEmailAddress(ctx context.Context, user *fleet.User, email string, password *string) error {
	// password requirement handled in validation middleware
	if password != nil {
		err := user.ValidatePassword(*password)
		if err != nil {
			return fleet.NewPermissionError("incorrect password")
		}
	}
	random, err := server.GenerateRandomText(svc.config.App.TokenKeySize)
	if err != nil {
		return err
	}
	token := base64.URLEncoding.EncodeToString([]byte(random))
	err = svc.ds.PendingEmailChange(ctx, user.ID, email, token)
	if err != nil {
		return err
	}
	config, err := svc.AppConfig(ctx)
	if err != nil {
		return err
	}

	changeEmail := fleet.Email{
		Subject: "Confirm Fleet Email Change",
		To:      []string{email},
		Config:  config,
		Mailer: &mail.ChangeEmailMailer{
			Token:    token,
			BaseURL:  template.URL(config.ServerSettings.ServerURL + svc.config.Server.URLPrefix),
			AssetURL: getAssetURL(),
		},
	}
	return svc.mailService.SendEmail(changeEmail)
}

// saves user in datastore.
// doesn't need to be exposed to the transport
// the service should expose actions for modifying a user instead
func (svc *Service) saveUser(ctx context.Context, user *fleet.User) error {
	return svc.ds.SaveUser(ctx, user)
}
