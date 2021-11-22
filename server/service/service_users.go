package service

import (
	"context"
	"encoding/base64"
	"html/template"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mail"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func (svc *Service) CreateUserFromInvite(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	// skipauth: There is no viewer context at this point. We rely on verifying
	// the invite for authNZ.
	svc.authz.SkipAuthorization(ctx)

	invite, err := svc.VerifyInvite(ctx, *p.InviteToken)
	if err != nil {
		return nil, err
	}

	// set the payload role property based on an existing invite.
	p.GlobalRole = invite.GlobalRole.Ptr()
	p.Teams = &invite.Teams

	user, err := svc.newUser(ctx, p)
	if err != nil {
		return nil, err
	}

	err = svc.ds.DeleteInvite(ctx, invite.ID)
	if err != nil {
		return nil, err
	}
	return user, nil
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

func (svc *Service) CreateInitialUser(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	// skipauth: Only the initial user creation should be allowed to skip
	// authorization (because there is not yet a user context to check against).
	svc.authz.SkipAuthorization(ctx)

	setupRequired, err := svc.SetupRequired(ctx)
	if err != nil {
		return nil, err
	}
	if !setupRequired {
		return nil, ctxerr.New(ctx, "a user already exists")
	}

	// Initial user should be global admin with no explicit teams
	p.GlobalRole = ptr.String(fleet.RoleAdmin)
	p.Teams = nil

	return svc.newUser(ctx, p)
}

func (svc *Service) newUser(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	var ssoEnabled bool
	// if user is SSO generate a fake password
	if (p.SSOInvite != nil && *p.SSOInvite) || (p.SSOEnabled != nil && *p.SSOEnabled) {
		fakePassword, err := server.GenerateRandomText(14)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "generate stand-in password")
		}
		p.Password = &fakePassword
		ssoEnabled = true
	}
	user, err := p.User(svc.config.Auth.SaltKeySize, svc.config.Auth.BcryptCost)
	if err != nil {
		return nil, err
	}
	user.SSOEnabled = ssoEnabled
	user, err = svc.ds.NewUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return user, nil
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

func (svc *Service) DeleteUser(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.User{ID: id}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DeleteUser(ctx, id)
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

func (svc *Service) User(ctx context.Context, id uint) (*fleet.User, error) {
	if err := svc.authz.Authorize(ctx, &fleet.User{ID: id}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.UserByID(ctx, id)
}

func (svc *Service) UserUnauthorized(ctx context.Context, id uint) (*fleet.User, error) {
	// Explicitly no authorization check. Should only be used by middleware.
	return svc.ds.UserByID(ctx, id)
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

func (svc *Service) ListUsers(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
	if err := svc.authz.Authorize(ctx, &fleet.User{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListUsers(ctx, opt)
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

func (svc *Service) ResetPassword(ctx context.Context, token, password string) error {
	// skipauth: No viewer context available. The user is locked out of their
	// account and authNZ is performed entirely by providing a valid password
	// reset token.
	svc.authz.SkipAuthorization(ctx)

	reset, err := svc.ds.FindPassswordResetByToken(ctx, token)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "looking up reset by token")
	}
	user, err := svc.ds.UserByID(ctx, reset.UserID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving user")
	}

	if user.SSOEnabled {
		return ctxerr.New(ctx, "password reset for single sign on user not allowed")
	}

	// prevent setting the same password
	if err := user.ValidatePassword(password); err == nil {
		return fleet.NewInvalidArgumentError("new_password", "cannot reuse old password")
	}

	err = svc.setNewPassword(ctx, user, password)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "setting new password")
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

func (svc *Service) PerformRequiredPasswordReset(ctx context.Context, password string) (*fleet.User, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	user := vc.User

	if err := svc.authz.Authorize(ctx, user, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if user.SSOEnabled {
		return nil, ctxerr.New(ctx, "password reset for single sign on user not allowed")
	}
	if !user.IsAdminForcedPasswordReset() {
		return nil, ctxerr.New(ctx, "user does not require password reset")
	}

	// prevent setting the same password
	if err := user.ValidatePassword(password); err == nil {
		return nil, fleet.NewInvalidArgumentError("new_password", "cannot reuse old password")
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
		ExpiresAt: time.Now().Add(time.Hour * 24),
		UserID:    user.ID,
		Token:     token,
	}
	_, err = svc.ds.NewPasswordResetRequest(ctx, request)
	if err != nil {
		return err
	}

	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}

	resetEmail := fleet.Email{
		Subject: "Reset Your Fleet Password",
		To:      []string{user.Email},
		Config:  config,
		Mailer: &mail.PasswordResetMailer{
			BaseURL:  template.URL(config.ServerSettings.ServerURL + svc.config.Server.URLPrefix),
			AssetURL: getAssetURL(),
			Token:    token,
		},
	}

	return svc.mailService.SendEmail(resetEmail)
}

// saves user in datastore.
// doesn't need to be exposed to the transport
// the service should expose actions for modifying a user instead
func (svc *Service) saveUser(ctx context.Context, user *fleet.User) error {
	return svc.ds.SaveUser(ctx, user)
}
