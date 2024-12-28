package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/go-kit/log/level"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
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
	// Token is only returned when creating API-only (non-SSO) users.
	Token *string `json:"token,omitempty"`
	Err   error   `json:"error,omitempty"`
}

func (r createUserResponse) error() error { return r.Err }

func createUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*createUserRequest)
	user, sessionKey, err := svc.CreateUser(ctx, req.UserPayload)
	if err != nil {
		return createUserResponse{Err: err}, nil
	}
	return createUserResponse{
		User:  user,
		Token: sessionKey,
	}, nil
}

var errMailerRequiredForMFA = badRequest("Email must be set up to enable Fleet MFA")

func (svc *Service) CreateUser(ctx context.Context, p fleet.UserPayload) (*fleet.User, *string, error) {
	var teams []fleet.UserTeam
	if p.Teams != nil {
		teams = *p.Teams
	}
	if err := svc.authz.Authorize(ctx, &fleet.User{Teams: teams}, fleet.ActionWrite); err != nil {
		return nil, nil, err
	}

	if err := p.VerifyAdminCreate(); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "verify user payload")
	}

	if teams != nil {
		// Validate that the teams exist
		teamsSummary, err := svc.ds.TeamsSummary(ctx)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "fetching teams in attempt to verify team exists")
		}
		teamIDs := map[uint]struct{}{}
		for _, team := range teamsSummary {
			teamIDs[team.ID] = struct{}{}
		}
		for _, userTeam := range teams {
			_, ok := teamIDs[userTeam.Team.ID]
			if !ok {
				return nil, nil, ctxerr.Wrap(
					ctx, fleet.NewInvalidArgumentError("teams.id", fmt.Sprintf("team with id %d does not exist", userTeam.Team.ID)),
				)
			}
		}
	}

	if invite, err := svc.ds.InviteByEmail(ctx, *p.Email); err == nil && invite != nil {
		return nil, nil, ctxerr.Errorf(ctx, "%s already invited", *p.Email)
	}

	if p.AdminForcedPasswordReset == nil {
		// By default, force password reset for users created this way.
		p.AdminForcedPasswordReset = ptr.Bool(true)
	}

	// make sure we can send email before requiring email sending to log in
	if p.MFAEnabled != nil && *p.MFAEnabled {
		config, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return nil, nil, err
		}

		var smtpSettings fleet.SMTPSettings
		if config.SMTPSettings != nil {
			smtpSettings = *config.SMTPSettings
		}

		if !svc.mailService.CanSendEmail(smtpSettings) {
			return nil, nil, errMailerRequiredForMFA
		}
	}

	user, err := svc.NewUser(ctx, p)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "create user")
	}

	// The sessionKey is returned for API-only non-SSO users only.
	var sessionKey *string
	if user.APIOnly && !user.SSOEnabled {
		if p.Password == nil {
			// Should not happen but let's log just in case.
			level.Error(svc.logger).Log("err", err, "msg", "password not set during admin user creation")
		} else {
			// Create a session for the API-only user by logging in.
			_, session, err := svc.Login(ctx, user.Email, *p.Password, false)
			if err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "create session for api-only user")
			}
			sessionKey = &session.Key
		}
	}

	return user, sessionKey, nil
}

////////////////////////////////////////////////////////////////////////////////
// Create User From Invite
////////////////////////////////////////////////////////////////////////////////

func createUserFromInviteEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*createUserRequest)
	user, err := svc.CreateUserFromInvite(ctx, req.UserPayload)
	if err != nil {
		return createUserResponse{Err: err}, nil
	}
	return createUserResponse{User: user}, nil
}

func (svc *Service) CreateUserFromInvite(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	// skipauth: There is no viewer context at this point. We rely on verifying
	// the invite for authNZ.
	svc.authz.SkipAuthorization(ctx)

	if err := p.VerifyInviteCreate(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "verify user payload")
	}

	invite, err := svc.VerifyInvite(ctx, *p.InviteToken)
	if err != nil {
		return nil, err
	}

	// set the payload role property based on an existing invite.
	p.GlobalRole = invite.GlobalRole.Ptr()
	p.Teams = &invite.Teams
	p.MFAEnabled = ptr.Bool(invite.MFAEnabled)

	user, err := svc.NewUser(ctx, p)
	if err != nil {
		return nil, err
	}

	err = svc.ds.DeleteInvite(ctx, invite.ID)
	if err != nil {
		return nil, err
	}
	return user, nil
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

func listUsersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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
	user := &fleet.User{}
	if opt.TeamID != 0 {
		user.Teams = []fleet.UserTeam{{Team: fleet.Team{ID: opt.TeamID}}}
	}
	if err := svc.authz.Authorize(ctx, user, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListUsers(ctx, opt)
}

// //////////////////////////////////////////////////////////////////////////////
// Me (get own current user)
// //////////////////////////////////////////////////////////////////////////////
type getMeRequest struct {
	includeSettings bool `url:"include_settings"`
}

func meEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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
	userSettings, err := svc.GetUserSettings(ctx, user.ID)
	if err != nil {
		return getUserResponse{Err: err}, nil
	}
	return getUserResponse{User: user, AvailableTeams: availableTeams, Settings: userSettings}, nil
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
	ID              uint `url:"id"`
	IncludeSettings bool `url"include_settings"`
}

type getUserResponse struct {
	User           *fleet.User          `json:"user,omitempty"`
	AvailableTeams []*fleet.TeamSummary `json:"available_teams"`
	Settings       *fleet.UserSettings  `json:"settings,omitempty"`
	Err            error                `json:"error,omitempty"`
}

func (r getUserResponse) error() error { return r.Err }

func getUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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

	userSettings, err := svc.GetUserSettings(ctx, user.ID)
	if err != nil {
		return getUserResponse{Err: err}, nil
	}
	return getUserResponse{User: user, AvailableTeams: availableTeams, Settings: userSettings}, nil
}

func (svc *Service) GetUserSettings(ctx context.Context, userID uint) (*fleet.UserSettings, error) {
	if err := svc.authz.Authorize(ctx, &fleet.User{ID: userID}, fleet.ActionRead); err != nil {
		return nil, err
	}
	return svc.ds.UserSettings(ctx, userID)
}

// setAuthCheckedOnPreAuthErr can be used to set the authentication as checked
// in case of errors that happened before an auth check can be performed.
// Otherwise the endpoints return a "authentication skipped" error instead of
// the actual returned error.
func setAuthCheckedOnPreAuthErr(ctx context.Context) {
	if az, ok := authz_ctx.FromContext(ctx); ok {
		az.SetChecked()
	}
}

func (svc *Service) User(ctx context.Context, id uint) (*fleet.User, error) {
	user, err := svc.ds.UserByID(ctx, id)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, ctxerr.Wrap(ctx, err)
	}

	if err := svc.authz.Authorize(ctx, user, fleet.ActionRead); err != nil {
		return nil, err
	}
	return user, nil
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

func modifyUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*modifyUserRequest)
	user, err := svc.ModifyUser(ctx, req.ID, req.UserPayload)
	if err != nil {
		return modifyUserResponse{Err: err}, nil
	}

	return modifyUserResponse{User: user}, nil
}

func (svc *Service) ModifyUser(ctx context.Context, userID uint, p fleet.UserPayload) (*fleet.User, error) {
	user, err := svc.User(ctx, userID)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, err
	}

	oldGlobalRole := user.GlobalRole
	oldTeams := user.Teams

	if err := svc.authz.Authorize(ctx, user, fleet.ActionWrite); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, ctxerr.New(ctx, "viewer not present") // should never happen, authorize would've failed
	}
	ownUser := vc.UserID() == userID
	if err := p.VerifyModify(ownUser); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "verify user payload")
	}

	if p.MFAEnabled != nil {
		if *p.MFAEnabled && !user.MFAEnabled {
			lic, _ := license.FromContext(ctx)
			if lic == nil {
				return nil, ctxerr.New(ctx, "license not found")
			}
			if !lic.IsPremium() {
				return nil, fleet.ErrMissingLicense
			}
			if (p.SSOEnabled != nil && *p.SSOEnabled) || (p.SSOEnabled == nil && user.SSOEnabled) {
				return nil, SSOMFAConflict
			}

			// make sure we can send email before requiring email sending to log in
			config, err := svc.ds.AppConfig(ctx)
			if err != nil {
				return nil, err
			}

			var smtpSettings fleet.SMTPSettings
			if config.SMTPSettings != nil {
				smtpSettings = *config.SMTPSettings
			}

			if !svc.mailService.CanSendEmail(smtpSettings) {
				return nil, errMailerRequiredForMFA
			}
		}
		user.MFAEnabled = *p.MFAEnabled
	}

	if (p.SSOEnabled != nil && *p.SSOEnabled) && user.MFAEnabled {
		return nil, SSOMFAConflict
	}

	if p.GlobalRole != nil || p.Teams != nil {
		if err := svc.authz.Authorize(ctx, user, fleet.ActionWriteRole); err != nil {
			return nil, err
		}
		license, _ := license.FromContext(ctx)
		if license == nil {
			return nil, ctxerr.New(ctx, "license not found")
		}
		if err := fleet.ValidateUserRoles(false, p, *license); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "validate role")
		}
	}

	if p.NewPassword != nil {
		if err := svc.authz.Authorize(ctx, user, fleet.ActionChangePassword); err != nil {
			return nil, err
		}
		if err := fleet.ValidatePasswordRequirements(*p.NewPassword); err != nil {
			return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("new_password", err.Error()))
		}
		if ownUser {
			// when changing one's own password, user cannot reuse the same password
			// and the old password must be provided (validated by p.VerifyModify above)
			// and must be valid. If changed by admin, then this is not required.
			if err := vc.User.ValidatePassword(*p.NewPassword); err == nil {
				return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("new_password", "Cannot reuse old password"))
			}
			if err := vc.User.ValidatePassword(*p.Password); err != nil {
				return nil, ctxerr.Wrap(ctx, fleet.NewPermissionError("incorrect password"))
			}
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
			return nil, authz.ForbiddenWithInternal(
				"cannot edit global role as a team member",
				currentUser, user, fleet.ActionWriteRole,
			)
		}

		if p.Teams != nil && len(*p.Teams) > 0 {
			return nil, fleet.NewInvalidArgumentError("teams", "may not be specified with global_role")
		}
		user.GlobalRole = p.GlobalRole
		user.Teams = []fleet.UserTeam{}
	} else if p.Teams != nil {
		if !isAdminOfTheModifiedTeams(currentUser, user.Teams, *p.Teams) {
			return nil, authz.ForbiddenWithInternal(
				"cannot modify teams in that way",
				currentUser, user, fleet.ActionWriteRole,
			)
		}
		user.Teams = *p.Teams
		user.GlobalRole = nil
	}

	if p.NewPassword != nil {
		// setNewPassword takes care of calling saveUser
		err = svc.setNewPassword(ctx, user, *p.NewPassword)
	} else {
		err = svc.saveUser(ctx, user)
	}
	if err != nil {
		return nil, err
	}

	// Load user again to get team-details like names.
	// Since we just modified the user and the changes may not have replicated to the read replica(s) yet,
	// we must use the master to ensure we get the most up-to-date information.
	ctxUsePrimary := ctxdb.RequirePrimary(ctx, true)
	user, err = svc.User(ctxUsePrimary, userID)
	if err != nil {
		return nil, err
	}
	adminUser := authz.UserFromContext(ctx)
	if err := fleet.LogRoleChangeActivities(ctx, svc, adminUser, oldGlobalRole, oldTeams, user); err != nil {
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

func deleteUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteUserRequest)
	err := svc.DeleteUser(ctx, req.ID)
	if err != nil {
		return deleteUserResponse{Err: err}, nil
	}
	return deleteUserResponse{}, nil
}

func (svc *Service) DeleteUser(ctx context.Context, id uint) error {
	user, err := svc.ds.UserByID(ctx, id)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return ctxerr.Wrap(ctx, err)
	}
	if err := svc.authz.Authorize(ctx, user, fleet.ActionWrite); err != nil {
		return err
	}
	if err := svc.ds.DeleteUser(ctx, id); err != nil {
		return err
	}

	adminUser := authz.UserFromContext(ctx)
	if err := svc.NewActivity(
		ctx,
		adminUser,
		fleet.ActivityTypeDeletedUser{
			UserID:    user.ID,
			UserName:  user.Name,
			UserEmail: user.Email,
		},
	); err != nil {
		return err
	}

	return nil
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

func requirePasswordResetEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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

func changePasswordEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*changePasswordRequest)
	err := svc.ChangePassword(ctx, req.OldPassword, req.NewPassword)
	return changePasswordResponse{Err: err}, nil
}

func (svc *Service) ChangePassword(ctx context.Context, oldPass, newPass string) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	if err := svc.authz.Authorize(ctx, vc.User, fleet.ActionChangePassword); err != nil {
		return err
	}

	if oldPass == "" {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("old_password", "Old password cannot be empty"))
	}
	if newPass == "" {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("new_password", "New password cannot be empty"))
	}
	if err := fleet.ValidatePasswordRequirements(newPass); err != nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("new_password", err.Error()))
	}
	if vc.User.SSOEnabled {
		return ctxerr.New(ctx, "change password for single sign on user not allowed")
	}
	if err := vc.User.ValidatePassword(newPass); err == nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("new_password", "Cannot reuse old password"))
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

func getInfoAboutSessionsForUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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
	if err := svc.authz.Authorize(ctx, &fleet.Session{UserID: id}, fleet.ActionRead); err != nil {
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

func deleteSessionsForUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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

////////////////////////////////////////////////////////////////////////////////
// Change user email
////////////////////////////////////////////////////////////////////////////////

type changeEmailRequest struct {
	Token string `url:"token"`
}

type changeEmailResponse struct {
	NewEmail string `json:"new_email"`
	Err      error  `json:"error,omitempty"`
}

func (r changeEmailResponse) error() error { return r.Err }

func changeEmailEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*changeEmailRequest)
	newEmailAddress, err := svc.ChangeUserEmail(ctx, req.Token)
	if err != nil {
		return changeEmailResponse{Err: err}, nil
	}
	return changeEmailResponse{NewEmail: newEmailAddress}, nil
}

func (svc *Service) ChangeUserEmail(ctx context.Context, token string) (string, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return "", fleet.ErrNoContext
	}

	if err := svc.authz.Authorize(ctx, &fleet.User{ID: vc.UserID()}, fleet.ActionWrite); err != nil {
		return "", err
	}

	return svc.ds.ConfirmPendingEmailChange(ctx, vc.UserID(), token)
}

// isAdminOfTheModifiedTeams checks whether the current user is allowed to modify the user
// roles in the teams.
//
// TODO: End-goal is to move all this logic to policy.rego.
func isAdminOfTheModifiedTeams(currentUser *fleet.User, originalUserTeams, newUserTeams []fleet.UserTeam) bool {
	// Global admins can modify all user teams roles.
	if currentUser.GlobalRole != nil && *currentUser.GlobalRole == fleet.RoleAdmin {
		return true
	}

	// Otherwise, make a map of the original and resulting teams.
	newTeams := make(map[uint]string)
	for _, team := range newUserTeams {
		newTeams[team.ID] = team.Role
	}
	originalTeams := make(map[uint]struct{})
	for _, team := range originalUserTeams {
		originalTeams[team.ID] = struct{}{}
	}

	// See which ones were removed or changed from the original.
	teamsAffected := make(map[uint]struct{})
	for _, team := range originalUserTeams {
		if newTeams[team.ID] != team.Role {
			teamsAffected[team.ID] = struct{}{}
		}
	}

	// See which ones of the new are not in the original.
	for _, team := range newUserTeams {
		if _, ok := originalTeams[team.ID]; !ok {
			teamsAffected[team.ID] = struct{}{}
		}
	}

	// Then gather the teams the current user is admin for.
	currentUserTeamAdmin := make(map[uint]struct{})
	for _, team := range currentUser.Teams {
		if team.Role == fleet.RoleAdmin {
			currentUserTeamAdmin[team.ID] = struct{}{}
		}
	}

	// And finally, let's check that the teams that were either removed
	// or changed are also teams this user is an admin of.
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

	switch _, err = svc.ds.UserByEmail(ctx, email); {
	case err == nil:
		return ctxerr.Wrap(ctx, newAlreadyExistsError())
	case errors.Is(err, sql.ErrNoRows):
		// OK
	default:
		return ctxerr.Wrap(ctx, err)
	}

	switch _, err = svc.ds.InviteByEmail(ctx, email); {
	case err == nil:
		return ctxerr.Wrap(ctx, newAlreadyExistsError())
	case errors.Is(err, sql.ErrNoRows):
		// OK
	default:
		return ctxerr.Wrap(ctx, err)
	}

	err = svc.ds.PendingEmailChange(ctx, user.ID, email, token)
	if err != nil {
		return err
	}
	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}

	var smtpSettings fleet.SMTPSettings
	if config.SMTPSettings != nil {
		smtpSettings = *config.SMTPSettings
	}

	changeEmail := fleet.Email{
		Subject:      "Confirm Fleet Email Change",
		To:           []string{email},
		SMTPSettings: smtpSettings,
		ServerURL:    config.ServerSettings.ServerURL,
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

////////////////////////////////////////////////////////////////////////////////
// Perform Required Password Reset
////////////////////////////////////////////////////////////////////////////////

type performRequiredPasswordResetRequest struct {
	Password string `json:"new_password"`
	ID       uint   `json:"id"`
}

type performRequiredPasswordResetResponse struct {
	User *fleet.User `json:"user,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r performRequiredPasswordResetResponse) error() error { return r.Err }

func performRequiredPasswordResetEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*performRequiredPasswordResetRequest)
	user, err := svc.PerformRequiredPasswordReset(ctx, req.Password)
	if err != nil {
		return performRequiredPasswordResetResponse{Err: err}, nil
	}
	return performRequiredPasswordResetResponse{User: user}, nil
}

func (svc *Service) PerformRequiredPasswordReset(ctx context.Context, password string) (*fleet.User, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		// No user in the context -- authentication issue
		svc.authz.SkipAuthorization(ctx)
		return nil, authz.ForbiddenWithInternal("No user in the context", nil, nil, nil)
	}
	if !vc.CanPerformPasswordReset() {
		svc.authz.SkipAuthorization(ctx)
		return nil, fleet.NewPermissionError("cannot reset password")
	}
	user := vc.User

	if err := svc.authz.Authorize(ctx, user, fleet.ActionChangePassword); err != nil {
		return nil, err
	}

	if user.SSOEnabled {
		// should never happen because this would get caught by the
		// CanPerformPasswordReset check above
		err := fleet.NewPermissionError("password reset for single sign on user not allowed")
		return nil, ctxerr.Wrap(ctx, err)
	}
	if !user.IsAdminForcedPasswordReset() {
		// should never happen because this would get caught by the
		// CanPerformPasswordReset check above
		err := fleet.NewPermissionError("cannot reset password")
		return nil, ctxerr.Wrap(ctx, err)
	}

	// prevent setting the same password
	if err := user.ValidatePassword(password); err == nil {
		return nil, fleet.NewInvalidArgumentError("new_password", "Cannot reuse old password")
	}

	if err := fleet.ValidatePasswordRequirements(password); err != nil {
		return nil, fleet.NewInvalidArgumentError("new_password", "Password does not meet required criteria: Must include 12 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#).")
	}

	user.AdminForcedPasswordReset = false
	err := svc.setNewPassword(ctx, user, password)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "setting new password")
	}

	// Sessions should already have been cleared when the reset was
	// required

	return user, nil
}

// setNewPassword is a helper for changing a user's password. It should be
// called to set the new password after proper authorization has been
// performed.
func (svc *Service) setNewPassword(ctx context.Context, user *fleet.User, password string) error {
	err := user.SetPassword(password, svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "setting new password")
	}
	if user.SSOEnabled {
		return ctxerr.New(ctx, "set password for single sign on user not allowed")
	}
	err = svc.saveUser(ctx, user)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "saving changed password")
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Reset Password
////////////////////////////////////////////////////////////////////////////////

type resetPasswordRequest struct {
	PasswordResetToken string `json:"password_reset_token"`
	NewPassword        string `json:"new_password"`
}

type resetPasswordResponse struct {
	Err error `json:"error,omitempty"`
}

func (r resetPasswordResponse) error() error { return r.Err }

func resetPasswordEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*resetPasswordRequest)
	err := svc.ResetPassword(ctx, req.PasswordResetToken, req.NewPassword)
	return resetPasswordResponse{Err: err}, nil
}

func (svc *Service) ResetPassword(ctx context.Context, token, password string) error {
	// skipauth: No viewer context available. The user is locked out of their
	// account and authNZ is performed entirely by providing a valid password
	// reset token.
	svc.authz.SkipAuthorization(ctx)

	if token == "" {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("token", "Token cannot be empty field"))
	}
	if password == "" {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("new_password", "New password cannot be empty field"))
	}
	if err := fleet.ValidatePasswordRequirements(password); err != nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("new_password", err.Error()))
	}

	reset, err := svc.ds.FindPasswordResetByToken(ctx, token)
	if err != nil {
		return ctxerr.Wrap(ctx, fleet.NewAuthFailedError(err.Error()), "find password reset request by token")
	}
	user, err := svc.ds.UserByID(ctx, reset.UserID)
	if err != nil {
		return ctxerr.Wrap(ctx, fleet.NewAuthFailedError(err.Error()), "find user by id")
	}

	if user.SSOEnabled {
		return ctxerr.New(ctx, "password reset for single sign on user not allowed")
	}

	// prevent setting the same password
	if err := user.ValidatePassword(password); err == nil {
		return fleet.NewInvalidArgumentError("new_password", "Cannot reuse old password")
	}

	// password requirements are validated as part of `setNewPassword``
	err = svc.setNewPassword(ctx, user, password)
	if err != nil {
		return fleet.NewInvalidArgumentError("new_password", err.Error())
	}

	// delete password reset tokens for user
	if err := svc.ds.DeletePasswordResetRequestsForUser(ctx, user.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete password reset requests")
	}

	// Clear sessions so that any other browsers will have to log in with
	// the new password
	if err := svc.ds.DestroyAllSessionsForUser(ctx, user.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete user sessions")
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Forgot Password
////////////////////////////////////////////////////////////////////////////////

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type forgotPasswordResponse struct {
	Err error `json:"error,omitempty"`
}

func (r forgotPasswordResponse) error() error { return r.Err }
func (r forgotPasswordResponse) Status() int  { return http.StatusAccepted }

func forgotPasswordEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*forgotPasswordRequest)
	// Any error returned by the service should not be returned to the
	// client to prevent information disclosure (it will be logged in the
	// server logs).
	_ = svc.RequestPasswordReset(ctx, req.Email)
	return forgotPasswordResponse{}, nil
}

func (svc *Service) RequestPasswordReset(ctx context.Context, email string) error {
	// skipauth: No viewer context available. The user is locked out of their
	// account and trying to reset their password.
	svc.authz.SkipAuthorization(ctx)

	// Regardless of error, sleep until the request has taken at least 1 second.
	// This means that any request to this method will take ~1s and frustrate a timing attack.
	defer func(start time.Time) {
		time.Sleep(time.Until(start.Add(1 * time.Second)))
	}(time.Now())

	user, err := svc.ds.UserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if user.SSOEnabled {
		return ctxerr.New(ctx, "password reset for single sign on user not allowed")
	}

	random, err := server.GenerateRandomText(svc.config.App.TokenKeySize)
	if err != nil {
		return err
	}
	token := base64.URLEncoding.EncodeToString([]byte(random))

	request := &fleet.PasswordResetRequest{
		UserID: user.ID,
		Token:  token,
	}
	_, err = svc.ds.NewPasswordResetRequest(ctx, request)
	if err != nil {
		return err
	}

	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}

	var smtpSettings fleet.SMTPSettings
	if config.SMTPSettings != nil {
		smtpSettings = *config.SMTPSettings
	}

	resetEmail := fleet.Email{
		Subject:      "Reset Your Fleet Password",
		To:           []string{user.Email},
		SMTPSettings: smtpSettings,
		ServerURL:    config.ServerSettings.ServerURL,
		Mailer: &mail.PasswordResetMailer{
			BaseURL:  template.URL(config.ServerSettings.ServerURL + svc.config.Server.URLPrefix),
			AssetURL: getAssetURL(),
			Token:    token,
		},
	}

	err = svc.mailService.SendEmail(resetEmail)
	if err != nil {
		level.Error(svc.logger).Log("err", err, "msg", "failed to send password reset request email")
	}
	return err
}

func (svc *Service) ListAvailableTeamsForUser(ctx context.Context, user *fleet.User) ([]*fleet.TeamSummary, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}
