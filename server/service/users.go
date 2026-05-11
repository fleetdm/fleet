package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	apiendpoints "github.com/fleetdm/fleet/v4/server/api_endpoints"
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

func (r createUserResponse) Error() error { return r.Err }

func createUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*createUserRequest)

	if req.APIEndpoints != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return createUserResponse{
			Err: fleet.NewInvalidArgumentError(
				"api_endpoints",
				"This endpoint does not accept API endpoint values",
			),
		}, nil
	}

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

func validateAPIEndpointRefs(ctx context.Context, refs *[]fleet.APIEndpointRef) error {
	if refs == nil {
		// Absent (nil pointer): no change.
		return nil
	}
	if *refs == nil {
		// Null (non-nil pointer to nil slice): clear all entries — full access.
		return nil
	}
	if len(*refs) == 0 {
		// Explicit empty array: not valid; send null to grant full access.
		return ctxerr.Wrap(
			ctx,
			fleet.NewInvalidArgumentError(
				"api_endpoints",
				"at least one API endpoint must be specified",
			),
		)
	}

	allEndpoints := apiendpoints.GetAPIEndpoints()
	entries := *refs

	if len(entries) > len(allEndpoints) {
		return ctxerr.Wrap(
			ctx,
			fleet.NewInvalidArgumentError("api_endpoints", "maximum number of API endpoints reached"),
		)
	}

	fpMap := make(map[string]struct{}, len(allEndpoints))
	for _, ep := range allEndpoints {
		fpMap[ep.Fingerprint()] = struct{}{}
	}
	seen := make(map[string]struct{}, len(entries))
	hasDuplicates := false
	var unknownFps []string
	for _, ref := range entries {
		fp := fleet.NewAPIEndpointFromTpl(ref.Method, ref.Path).Fingerprint()
		if _, dup := seen[fp]; dup {
			hasDuplicates = true
			continue
		}
		seen[fp] = struct{}{}
		if _, ok := fpMap[fp]; !ok {
			unknownFps = append(unknownFps, fp)
		}
	}
	invalid := &fleet.InvalidArgumentError{}
	if hasDuplicates {
		invalid.Append("api_endpoints", "one or more api_endpoints entries are duplicated")
	}
	if len(unknownFps) > 0 {
		invalid.Append("api_endpoints", fmt.Sprintf("one or more api_endpoints entries are invalid: %s", strings.Join(unknownFps, ", ")))
	}
	if invalid.HasErrors() {
		return ctxerr.Wrap(ctx, invalid, "validate api_endpoints")
	}
	return nil
}

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

	// Do not allow creating a user with any Premium-only features on Fleet Free.
	if !license.IsPremium(ctx) {
		var teamRoles []fleet.UserTeam
		if p.Teams != nil {
			teamRoles = *p.Teams
		}
		if fleet.PremiumRolesPresent(p.GlobalRole, teamRoles) {
			return nil, nil, fleet.ErrMissingLicense
		}
		if p.APIOnly != nil && *p.APIOnly && p.APIEndpoints != nil && *p.APIEndpoints != nil {
			return nil, nil, fleet.ErrMissingLicense
		}
		if p.APIOnly != nil && *p.APIOnly && len(teamRoles) > 0 {
			return nil, nil, fleet.ErrMissingLicense
		}
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
					ctx, fleet.NewInvalidArgumentError("teams.id", fmt.Sprintf("fleet with id %d does not exist", userTeam.Team.ID)),
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

	if p.APIEndpoints != nil && (p.APIOnly == nil || !*p.APIOnly) {
		return nil, nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("api_endpoints", "API endpoints can only be specified for API only users"))
	}
	if p.APIOnly != nil && *p.APIOnly {
		if err := validateAPIEndpointRefs(ctx, p.APIEndpoints); err != nil {
			return nil, nil, err
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
			svc.logger.ErrorContext(ctx, "password not set during admin user creation")
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
// Create API-Only user
////////////////////////////////////////////////////////////////////////////////

type fleetsPayload struct {
	ID   uint   `json:"id" db:"id"`
	Role string `json:"role" db:"role"`
}

type createAPIOnlyUserRequest struct {
	Name         *string                 `json:"name,omitempty"`
	GlobalRole   *string                 `json:"global_role,omitempty"`
	Fleets       *[]fleetsPayload        `json:"fleets,omitempty"`
	APIEndpoints *[]fleet.APIEndpointRef `json:"api_endpoints,omitempty"`
}

func createAPIOnlyUserEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*createAPIOnlyUserRequest)

	pwd, err := server.GenerateRandomPwd()
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return createUserResponse{
			Err: ctxerr.Wrap(ctx, err, "generate user password"),
		}, nil
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		setAuthCheckedOnPreAuthErr(ctx)
		return createUserResponse{
			Err: ctxerr.New(ctx, "failed to get logged user"),
		}, nil
	}
	email, err := server.GenerateRandomEmail(vc.Email())
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return createUserResponse{
			Err: ctxerr.Wrap(ctx, err, "generate user email"),
		}, nil
	}

	var fleets []fleet.UserTeam
	if req.Fleets != nil {
		for _, t := range *req.Fleets {
			val := fleet.UserTeam{}
			val.ID = t.ID
			val.Role = t.Role
			fleets = append(fleets, val)
		}
	}

	user, token, err := svc.CreateUser(ctx, fleet.UserPayload{
		Name:                     req.Name,
		Email:                    &email,
		Password:                 &pwd,
		APIOnly:                  new(true),
		AdminForcedPasswordReset: new(false),
		GlobalRole:               req.GlobalRole,
		Teams:                    &fleets,
		APIEndpoints:             req.APIEndpoints,
	})
	if err != nil {
		return createUserResponse{Err: err}, nil
	}

	return createUserResponse{
		User:  user,
		Token: token,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Patch API-Only User
////////////////////////////////////////////////////////////////////////////////

type modifyAPIOnlyUserRequest struct {
	ID           uint                       `json:"-" url:"id"`
	Name         *string                    `json:"name,omitempty"`
	GlobalRole   *string                    `json:"global_role,omitempty"`
	Teams        *[]fleet.UserTeam          `json:"teams,omitempty" renameto:"fleets"`
	APIEndpoints fleet.OptionalAPIEndpoints `json:"api_endpoints"`
}

type modifyAPIOnlyUserResponse struct {
	User *fleet.User `json:"user,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r modifyAPIOnlyUserResponse) Error() error { return r.Err }

func modifyAPIOnlyUserEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*modifyAPIOnlyUserRequest)

	payload := fleet.UserPayload{
		Name:       req.Name,
		GlobalRole: req.GlobalRole,
		Teams:      req.Teams,
	}
	if req.APIEndpoints.Present {
		if req.APIEndpoints.Value == nil {
			// null → clear all entries; signal via non-nil pointer to nil slice.
			var emptyEndpoints []fleet.APIEndpointRef
			payload.APIEndpoints = &emptyEndpoints
		} else {
			payload.APIEndpoints = &req.APIEndpoints.Value
		}
	}

	user, err := svc.ModifyAPIOnlyUser(ctx, req.ID, payload)
	if err != nil {
		return modifyAPIOnlyUserResponse{Err: err}, nil
	}
	return modifyAPIOnlyUserResponse{User: user}, nil
}

func (svc *Service) ModifyAPIOnlyUser(ctx context.Context, userID uint, p fleet.UserPayload) (*fleet.User, error) {
	target, err := svc.ds.UserByID(ctx, userID)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, ctxerr.Wrap(ctx, err)
	}
	if err := svc.authz.Authorize(ctx, target, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if !target.APIOnly {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("id", "target user is not an API-only user"))
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, ctxerr.New(ctx, "viewer not present")
	}
	if vc.UserID() == userID {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("id", "cannot modify your own API-only user"))
	}

	return svc.ModifyUser(ctx, userID, fleet.UserPayload{
		Name:         p.Name,
		GlobalRole:   p.GlobalRole,
		Teams:        p.Teams,
		APIOnly:      new(true),
		APIEndpoints: p.APIEndpoints,
	})
}

////////////////////////////////////////////////////////////////////////////////
// Create User From Invite
////////////////////////////////////////////////////////////////////////////////

func createUserFromInviteEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*createUserRequest)

	if req.APIOnly != nil || req.APIEndpoints != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return createUserResponse{
			Err: fleet.NewInvalidArgumentError(
				"api_endpoints",
				"This endpoint does not accept API endpoint values",
			),
		}, nil
	}

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

	var payloadEmail string
	if p.Email != nil {
		payloadEmail = *p.Email
	}
	if invite.Email != payloadEmail {
		return nil, fleet.NewInvalidArgumentError("invite_token", "Invite Token does not match Email Address.")
	}

	// set the payload role property based on an existing invite.
	p.GlobalRole = invite.GlobalRole.Ptr()
	p.Teams = &invite.Teams
	p.MFAEnabled = ptr.Bool(invite.MFAEnabled)
	// Invite ID is only used as a uniq index to prevent a double invite acceptance race condition
	p.InviteID = &invite.ID

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

func (r listUsersResponse) Error() error { return r.Err }

func listUsersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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

func (svc *Service) UsersByIDs(ctx context.Context, ids []uint) ([]*fleet.UserSummary, error) {
	// Authorize read access to users (no specific team context)
	if err := svc.authz.Authorize(ctx, &fleet.User{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.UsersByIDs(ctx, ids)
}

// //////////////////////////////////////////////////////////////////////////////
// Me (get own current user)
// //////////////////////////////////////////////////////////////////////////////
type getMeRequest struct {
	IncludeUISettings bool `query:"include_ui_settings,optional"`
}

func meEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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
	req := request.(*getMeRequest)
	var userSettings *fleet.UserSettings
	if req.IncludeUISettings {
		userSettings, err = svc.GetUserSettings(ctx, user.ID)
		if err != nil {
			return getUserResponse{Err: err}, nil
		}
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
	ID                uint `url:"id"`
	IncludeUISettings bool `query:"include_ui_settings,optional"`
}

type getUserResponse struct {
	User           *fleet.User          `json:"user,omitempty"`
	AvailableTeams []*fleet.TeamSummary `json:"available_teams" renameto:"available_fleets"`
	Settings       *fleet.UserSettings  `json:"settings,omitempty"`
	Err            error                `json:"error,omitempty"`
}

func (r getUserResponse) Error() error { return r.Err }

func getUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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

	var userSettings *fleet.UserSettings
	if req.IncludeUISettings {
		userSettings, err = svc.GetUserSettings(ctx, user.ID)
		if err != nil {
			return getUserResponse{Err: err}, nil
		}
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

func (r modifyUserResponse) Error() error { return r.Err }

func modifyUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*modifyUserRequest)

	if req.APIOnly != nil || req.APIEndpoints != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return modifyUserResponse{
			Err: fleet.NewInvalidArgumentError(
				"api_endpoints",
				"This endpoint does not accept API endpoint values",
			),
		}, nil
	}

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

	// Do not allow setting any Premium-only features on Fleet Free.
	if !license.IsPremium(ctx) {
		var teamRoles []fleet.UserTeam
		if p.Teams != nil {
			teamRoles = *p.Teams
		}
		if fleet.PremiumRolesPresent(p.GlobalRole, teamRoles) {
			return nil, fleet.ErrMissingLicense
		}
		if user.APIOnly && p.APIEndpoints != nil && *p.APIEndpoints != nil {
			return nil, fleet.ErrMissingLicense
		}
		if p.APIOnly != nil && *p.APIOnly && len(teamRoles) > 0 {
			return nil, fleet.ErrMissingLicense
		}
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, ctxerr.New(ctx, "viewer not present") // should never happen, authorize would've failed
	}
	ownUser := vc.UserID() == userID
	if err := p.VerifyModify(ownUser); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "verify user payload")
	}

	if p.APIOnly != nil && *p.APIOnly != user.APIOnly {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("api_only", "cannot change api_only status of a user"))
	}
	if p.APIEndpoints != nil && !user.APIOnly {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("api_endpoints", "API endpoints can only be specified for API only users"))
	}
	if p.APIEndpoints != nil {
		// Changing endpoint permissions is a privileged operation — same level as
		// changing roles. This prevents an API-only user from expanding their own access.
		if err := svc.authz.Authorize(ctx, user, fleet.ActionWriteRole); err != nil {
			return nil, err
		}
	}
	if err := validateAPIEndpointRefs(ctx, p.APIEndpoints); err != nil {
		return nil, err
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
		licChecker, _ := license.FromContext(ctx)
		lic, _ := licChecker.(*fleet.LicenseInfo)
		if lic == nil {
			return nil, ctxerr.New(ctx, "license not found")
		}
		if err := fleet.ValidateUserRoles(false, p, *lic); err != nil {
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
		if !*p.SSOEnabled && user.SSOEnabled && p.NewPassword == nil {
			return nil, fleet.NewInvalidArgumentError("missing password", "a new password must be provided when disabling SSO")
		}
		user.SSOEnabled = *p.SSOEnabled
	}

	if p.Settings != nil {
		user.Settings = p.Settings
	}

	if p.APIEndpoints != nil {
		user.APIEndpoints = *p.APIEndpoints
	}

	currentUser := authz.UserFromContext(ctx)

	var isGlobalAdminDemotion bool

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

		// Track whether this is a demotion from global admin so we can
		// use an atomic check+save later to prevent TOCTOU races.
		isGlobalAdminDemotion = user.GlobalRole != nil && *user.GlobalRole == fleet.RoleAdmin && *p.GlobalRole != fleet.RoleAdmin

		user.GlobalRole = p.GlobalRole
		user.Teams = []fleet.UserTeam{}
	} else if p.Teams != nil {
		// Track whether this is a demotion from global admin by assigning teams.
		isGlobalAdminDemotion = user.GlobalRole != nil && *user.GlobalRole == fleet.RoleAdmin

		if !isAdminOfTheModifiedTeams(currentUser, user.Teams, *p.Teams) {
			return nil, authz.ForbiddenWithInternal(
				"cannot modify teams in that way",
				currentUser, user, fleet.ActionWriteRole,
			)
		}
		user.Teams = *p.Teams
		user.GlobalRole = nil
	}

	switch {
	case isGlobalAdminDemotion:
		// Use atomic check+save to prevent TOCTOU race when demoting the last admin.
		// We must set the password before saving if a new password was also provided.
		if p.NewPassword != nil {
			if err = user.SetPassword(*p.NewPassword, svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "setting new password")
			}
		}
		err = svc.ds.SaveUserIfNotLastAdmin(ctx, user)
		if errors.Is(err, fleet.ErrLastGlobalAdmin) {
			if p.GlobalRole != nil {
				return nil, fleet.NewInvalidArgumentError("global_role", "cannot demote the last global admin")
			}
			return nil, fleet.NewInvalidArgumentError("teams", "cannot demote the last global admin")
		}
		if err == nil && p.NewPassword != nil {
			// Clean up password reset requests and sessions like setNewPassword does.
			if err := svc.ds.DeletePasswordResetRequestsForUser(ctx, user.ID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "deleting password reset requests after password change")
			}
			if err := svc.ds.DestroyAllSessionsForUser(ctx, user.ID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "destroying sessions after password change")
			}
		}
	case p.NewPassword != nil:
		// setNewPassword takes care of calling saveUser
		err = svc.setNewPassword(ctx, user, *p.NewPassword, true)
	default:
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

func (r deleteUserResponse) Error() error { return r.Err }

func deleteUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteUserRequest)
	if _, err := svc.DeleteUser(ctx, req.ID); err != nil {
		return deleteUserResponse{Err: err}, nil
	}
	return deleteUserResponse{}, nil
}

func (svc *Service) DeleteUser(ctx context.Context, id uint) (*fleet.User, error) {
	user, err := svc.ds.UserByID(ctx, id)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, ctxerr.Wrap(ctx, err)
	}
	if err := svc.authz.Authorize(ctx, user, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// Atomically check that we're not deleting the last global admin before deleting.
	if user.GlobalRole != nil && *user.GlobalRole == fleet.RoleAdmin {
		if err := svc.ds.DeleteUserIfNotLastAdmin(ctx, id); err != nil {
			if errors.Is(err, fleet.ErrLastGlobalAdmin) {
				return nil, fleet.NewInvalidArgumentError("id", "cannot delete the last global admin")
			}
			return nil, err
		}
	} else if err := svc.ds.DeleteUser(ctx, id); err != nil {
		return nil, err
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
		return nil, err
	}

	return user, nil
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

func (r requirePasswordResetResponse) Error() error { return r.Err }

func requirePasswordResetEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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
		// Clear all password reset tokens for good measure.
		if err := svc.ds.DeletePasswordResetRequestsForUser(ctx, user.ID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "deleting password reset requests after password change")
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

func (r changePasswordResponse) Error() error { return r.Err }

func changePasswordEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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

	if err := svc.setNewPassword(ctx, vc.User, newPass, true); err != nil {
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

func (r getInfoAboutSessionsForUserResponse) Error() error { return r.Err }

func getInfoAboutSessionsForUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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

func (r deleteSessionsForUserResponse) Error() error { return r.Err }

func deleteSessionsForUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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

func (r changeEmailResponse) Error() error { return r.Err }

func changeEmailEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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
	return svc.mailService.SendEmail(ctx, changeEmail)
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

func (r performRequiredPasswordResetResponse) Error() error { return r.Err }

func performRequiredPasswordResetEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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
	err := svc.setNewPassword(ctx, user, password, false)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "setting new password")
	}

	return user, nil
}

// setNewPassword is a helper for changing a user's password. It should be
// called to set the new password after proper authorization has been
// performed.
func (svc *Service) setNewPassword(ctx context.Context, user *fleet.User, password string, clearSessions bool) error {
	err := user.SetPassword(password, svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "setting new password")
	}
	err = svc.saveUser(ctx, user)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "saving changed password")
	}

	// Ensure that any existing links for password resets will no longer work.
	if err := svc.ds.DeletePasswordResetRequestsForUser(ctx, user.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting password reset requests after password change")
	}

	// Force the user to log in again with new password unless explicitly told not to.
	if clearSessions {
		if err := svc.ds.DestroyAllSessionsForUser(ctx, user.ID); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting sessions after password change")
		}
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

func (r resetPasswordResponse) Error() error { return r.Err }

func resetPasswordEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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
	err = svc.setNewPassword(ctx, user, password, true)
	if err != nil {
		return fleet.NewInvalidArgumentError("new_password", err.Error())
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

func (r forgotPasswordResponse) Error() error { return r.Err }
func (r forgotPasswordResponse) Status() int  { return http.StatusAccepted }

func forgotPasswordEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*forgotPasswordRequest)
	// Any error returned by the service should not be returned to the
	// client to prevent information disclosure (it will be logged in the
	// server logs).
	if err := svc.RequestPasswordReset(ctx, req.Email); errors.Is(err, fleet.ErrPasswordResetNotConfigured) {
		return forgotPasswordResponse{Err: err}, nil
	}
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

	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	if !svc.mailService.CanSendEmail(*config.SMTPSettings) {
		return fleet.ErrPasswordResetNotConfigured
	}

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

	err = svc.mailService.SendEmail(ctx, resetEmail)
	if err != nil {
		svc.logger.ErrorContext(ctx, "failed to send password reset request email", "err", err)
	}
	return err
}

func (svc *Service) ListAvailableTeamsForUser(ctx context.Context, user *fleet.User) ([]*fleet.TeamSummary, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}
