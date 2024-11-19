package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/go-kit/log/level"
)

func obfuscateSecrets(user *fleet.User, teams []*fleet.Team) error {
	if user == nil {
		return &authz.Forbidden{}
	}

	isGlobalObs := user.IsGlobalObserver()

	teamMemberships := user.TeamMembership(func(t fleet.UserTeam) bool {
		return true
	})
	obsMembership := user.TeamMembership(func(t fleet.UserTeam) bool {
		return t.Role == fleet.RoleObserver || t.Role == fleet.RoleObserverPlus
	})

	for _, t := range teams {
		if t == nil {
			continue
		}
		// User does not belong to the team or is a global/team observer/observer+
		if isGlobalObs || user.GlobalRole == nil && (!teamMemberships[t.ID] || obsMembership[t.ID]) {
			for _, s := range t.Secrets {
				s.Secret = fleet.MaskedPassword
			}
		}
	}
	return nil
}

func (svc *Service) NewTeam(ctx context.Context, p fleet.TeamPayload) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// Copy team options from global options
	globalConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, err
	}
	team := &fleet.Team{
		Config: fleet.TeamConfig{
			AgentOptions: globalConfig.AgentOptions,
			Features:     globalConfig.Features,
		},
	}

	if p.Name == nil {
		return nil, fleet.NewInvalidArgumentError("name", "missing required argument")
	}
	if *p.Name == "" {
		return nil, fleet.NewInvalidArgumentError("name", "may not be empty")
	}
	l := strings.ToLower(*p.Name)
	if l == strings.ToLower(fleet.ReservedNameAllTeams) {
		return nil, fleet.NewInvalidArgumentError("name", `"All teams" is a reserved team name`)
	}
	if l == strings.ToLower(fleet.ReservedNameNoTeam) {
		return nil, fleet.NewInvalidArgumentError("name", `"No team" is a reserved team name`)
	}
	team.Name = *p.Name

	if p.Description != nil {
		team.Description = *p.Description
	}

	if p.Secrets != nil {
		if len(p.Secrets) > fleet.MaxEnrollSecretsCount {
			return nil, fleet.NewInvalidArgumentError("secrets", "too many secrets")
		}
		team.Secrets = p.Secrets
	} else {
		// Set up a default enroll secret
		secret, err := server.GenerateRandomText(fleet.EnrollSecretDefaultLength)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "generate enroll secret string")
		}
		team.Secrets = []*fleet.EnrollSecret{{Secret: secret}}
	}

	if p.HostExpirySettings != nil && p.HostExpirySettings.HostExpiryEnabled && p.HostExpirySettings.HostExpiryWindow <= 0 {
		return nil, fleet.NewInvalidArgumentError("host_expiry_window", "must be greater than 0")
	}

	team, err = svc.ds.NewTeam(ctx, team)
	if err != nil {
		return nil, err
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeCreatedTeam{
			ID:   team.ID,
			Name: team.Name,
		},
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for team creation")
	}

	return team, nil
}

func (svc *Service) ModifyTeam(ctx context.Context, teamID uint, payload fleet.TeamPayload) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	team, err := svc.ds.Team(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if payload.Name != nil {
		if *payload.Name == "" {
			return nil, fleet.NewInvalidArgumentError("name", "may not be empty")
		}
		l := strings.ToLower(*payload.Name)
		if l == strings.ToLower(fleet.ReservedNameAllTeams) {
			return nil, fleet.NewInvalidArgumentError("name", `"All teams" is a reserved team name`)
		}
		if l == strings.ToLower(fleet.ReservedNameNoTeam) {
			return nil, fleet.NewInvalidArgumentError("name", `"No team" is a reserved team name`)
		}
		team.Name = *payload.Name
	}
	if payload.Description != nil {
		team.Description = *payload.Description
	}

	if payload.WebhookSettings != nil {
		team.Config.WebhookSettings = *payload.WebhookSettings
	}

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	var (
		macOSMinVersionUpdated        bool
		iOSMinVersionUpdated          bool
		iPadOSMinVersionUpdated       bool
		windowsUpdatesUpdated         bool
		macOSDiskEncryptionUpdated    bool
		macOSEnableEndUserAuthUpdated bool
	)
	if payload.MDM != nil {
		if payload.MDM.MacOSUpdates != nil {
			if err := payload.MDM.MacOSUpdates.Validate(); err != nil {
				return nil, fleet.NewInvalidArgumentError("macos_updates", err.Error())
			}
			if payload.MDM.MacOSUpdates.MinimumVersion.Set || payload.MDM.MacOSUpdates.Deadline.Set {
				macOSMinVersionUpdated = team.Config.MDM.MacOSUpdates.MinimumVersion.Value != payload.MDM.MacOSUpdates.MinimumVersion.Value ||
					team.Config.MDM.MacOSUpdates.Deadline.Value != payload.MDM.MacOSUpdates.Deadline.Value
				team.Config.MDM.MacOSUpdates = *payload.MDM.MacOSUpdates
			}
		}
		if payload.MDM.IOSUpdates != nil {
			if err := payload.MDM.IOSUpdates.Validate(); err != nil {
				return nil, fleet.NewInvalidArgumentError("ios_updates", err.Error())
			}
			if payload.MDM.IOSUpdates.MinimumVersion.Set || payload.MDM.IOSUpdates.Deadline.Set {
				iOSMinVersionUpdated = team.Config.MDM.IOSUpdates.MinimumVersion.Value != payload.MDM.IOSUpdates.MinimumVersion.Value ||
					team.Config.MDM.IOSUpdates.Deadline.Value != payload.MDM.IOSUpdates.Deadline.Value
				team.Config.MDM.IOSUpdates = *payload.MDM.IOSUpdates
			}
		}
		if payload.MDM.IPadOSUpdates != nil {
			if err := payload.MDM.IPadOSUpdates.Validate(); err != nil {
				return nil, fleet.NewInvalidArgumentError("ipados_updates", err.Error())
			}
			if payload.MDM.IPadOSUpdates.MinimumVersion.Set || payload.MDM.IPadOSUpdates.Deadline.Set {
				iPadOSMinVersionUpdated = team.Config.MDM.IPadOSUpdates.MinimumVersion.Value != payload.MDM.IPadOSUpdates.MinimumVersion.Value ||
					team.Config.MDM.IPadOSUpdates.Deadline.Value != payload.MDM.IPadOSUpdates.Deadline.Value
				team.Config.MDM.IPadOSUpdates = *payload.MDM.IPadOSUpdates
			}
		}

		if payload.MDM.WindowsUpdates != nil {
			if err := payload.MDM.WindowsUpdates.Validate(); err != nil {
				return nil, fleet.NewInvalidArgumentError("windows_updates", err.Error())
			}
			if payload.MDM.WindowsUpdates.DeadlineDays.Set || payload.MDM.WindowsUpdates.GracePeriodDays.Set {
				windowsUpdatesUpdated = !team.Config.MDM.WindowsUpdates.Equal(*payload.MDM.WindowsUpdates)
				team.Config.MDM.WindowsUpdates = *payload.MDM.WindowsUpdates
			}
		}

		if payload.MDM.EnableDiskEncryption.Valid {
			macOSDiskEncryptionUpdated = team.Config.MDM.EnableDiskEncryption != payload.MDM.EnableDiskEncryption.Value
			if macOSDiskEncryptionUpdated && !appCfg.MDM.EnabledAndConfigured {
				return nil, fleet.NewInvalidArgumentError("macos_settings.enable_disk_encryption",
					`Couldn't update macos_settings because MDM features aren't turned on in Fleet. Use fleetctl generate mdm-apple and then fleet serve with mdm configuration to turn on MDM features.`)
			}
			team.Config.MDM.EnableDiskEncryption = payload.MDM.EnableDiskEncryption.Value
		}

		if payload.MDM.MacOSSetup != nil {
			if !appCfg.MDM.EnabledAndConfigured && team.Config.MDM.MacOSSetup.EnableEndUserAuthentication != payload.MDM.MacOSSetup.EnableEndUserAuthentication {
				return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("macos_setup.enable_end_user_authentication",
					`Couldn't update macos_setup.enable_end_user_authentication because MDM features aren't turned on in Fleet. Use fleetctl generate mdm-apple and then fleet serve with mdm configuration to turn on MDM features.`))
			}
			macOSEnableEndUserAuthUpdated = team.Config.MDM.MacOSSetup.EnableEndUserAuthentication != payload.MDM.MacOSSetup.EnableEndUserAuthentication
			if macOSEnableEndUserAuthUpdated && payload.MDM.MacOSSetup.EnableEndUserAuthentication && appCfg.MDM.EndUserAuthentication.IsEmpty() {
				// TODO: update this error message to include steps to resolve the issue once docs for IdP
				// config are available
				return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("macos_setup.enable_end_user_authentication",
					`Couldn't enable macos_setup.enable_end_user_authentication because no IdP is configured for MDM features.`))
			}

			if err := svc.validateEndUserAuthenticationAndSetupAssistant(ctx, &team.ID); err != nil {
				return nil, err
			}

			team.Config.MDM.MacOSSetup.EnableEndUserAuthentication = payload.MDM.MacOSSetup.EnableEndUserAuthentication
		}
	}

	if payload.Integrations != nil {
		if payload.Integrations.Jira != nil || payload.Integrations.Zendesk != nil {
			// the team integrations must reference an existing global config integration.
			if _, err := payload.Integrations.MatchWithIntegrations(appCfg.Integrations); err != nil {
				return nil, fleet.NewInvalidArgumentError("integrations", err.Error())
			}

			// integrations must be unique
			if err := payload.Integrations.Validate(); err != nil {
				return nil, fleet.NewInvalidArgumentError("integrations", err.Error())
			}

			team.Config.Integrations.Jira = payload.Integrations.Jira
			team.Config.Integrations.Zendesk = payload.Integrations.Zendesk
		}
		// Only update the calendar integration if it's not nil
		if payload.Integrations.GoogleCalendar != nil {
			invalid := &fleet.InvalidArgumentError{}
			_ = svc.validateTeamCalendarIntegrations(payload.Integrations.GoogleCalendar, appCfg, false, invalid)
			if invalid.HasErrors() {
				return nil, ctxerr.Wrap(ctx, invalid)
			}
			team.Config.Integrations.GoogleCalendar = payload.Integrations.GoogleCalendar
		}
	}

	if payload.WebhookSettings != nil || payload.Integrations != nil {
		// must validate that at most only one automation is enabled for each
		// supported feature - by now the updated payload has been applied to
		// team.Config.
		invalid := &fleet.InvalidArgumentError{}
		fleet.ValidateEnabledFailingPoliciesTeamIntegrations(
			team.Config.WebhookSettings.FailingPoliciesWebhook,
			team.Config.Integrations,
			invalid,
		)
		if invalid.HasErrors() {
			return nil, ctxerr.Wrap(ctx, invalid)
		}
	}

	if payload.HostExpirySettings != nil {
		if payload.HostExpirySettings.HostExpiryEnabled && payload.HostExpirySettings.HostExpiryWindow <= 0 {
			return nil, fleet.NewInvalidArgumentError("host_expiry_window", "must be greater than 0")
		}
		team.Config.HostExpirySettings = *payload.HostExpirySettings
	}

	team, err = svc.ds.SaveTeam(ctx, team)
	if err != nil {
		return nil, err
	}

	if macOSMinVersionUpdated {
		if err := svc.mdmAppleEditedAppleOSUpdates(ctx, &team.ID, fleet.MacOS, team.Config.MDM.MacOSUpdates); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "update DDM profile on macOS updates change")
		}

		if err := svc.NewActivity(
			ctx,
			authz.UserFromContext(ctx),
			fleet.ActivityTypeEditedMacOSMinVersion{
				TeamID:         &team.ID,
				TeamName:       &team.Name,
				MinimumVersion: team.Config.MDM.MacOSUpdates.MinimumVersion.Value,
				Deadline:       team.Config.MDM.MacOSUpdates.Deadline.Value,
			},
		); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create activity for team macOS min version edited")
		}
	}
	if iOSMinVersionUpdated {
		if err := svc.mdmAppleEditedAppleOSUpdates(ctx, &team.ID, fleet.IOS, team.Config.MDM.IOSUpdates); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "update DDM profile on iOS updates change")
		}

		if err := svc.NewActivity(
			ctx,
			authz.UserFromContext(ctx),
			fleet.ActivityTypeEditedIOSMinVersion{
				TeamID:         &team.ID,
				TeamName:       &team.Name,
				MinimumVersion: team.Config.MDM.IOSUpdates.MinimumVersion.Value,
				Deadline:       team.Config.MDM.IOSUpdates.Deadline.Value,
			},
		); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create activity for team iOS min version edited")
		}
	}
	if iPadOSMinVersionUpdated {
		if err := svc.mdmAppleEditedAppleOSUpdates(ctx, &team.ID, fleet.IPadOS, team.Config.MDM.IPadOSUpdates); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "update DDM profile on iPadOS updates change")
		}

		if err := svc.NewActivity(
			ctx,
			authz.UserFromContext(ctx),
			fleet.ActivityTypeEditedIPadOSMinVersion{
				TeamID:         &team.ID,
				TeamName:       &team.Name,
				MinimumVersion: team.Config.MDM.IPadOSUpdates.MinimumVersion.Value,
				Deadline:       team.Config.MDM.IPadOSUpdates.Deadline.Value,
			},
		); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create activity for team iPadOS min version edited")
		}
	}

	if windowsUpdatesUpdated {
		var deadline, grace *int
		if team.Config.MDM.WindowsUpdates.DeadlineDays.Valid {
			deadline = &team.Config.MDM.WindowsUpdates.DeadlineDays.Value
		}
		if team.Config.MDM.WindowsUpdates.GracePeriodDays.Valid {
			grace = &team.Config.MDM.WindowsUpdates.GracePeriodDays.Value
		}

		if deadline != nil {
			if err := svc.mdmWindowsEnableOSUpdates(ctx, &team.ID, team.Config.MDM.WindowsUpdates); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "enable team windows OS updates")
			}
		} else if err := svc.mdmWindowsDisableOSUpdates(ctx, &team.ID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "disable team windows OS updates")
		}

		if err := svc.NewActivity(
			ctx,
			authz.UserFromContext(ctx),
			fleet.ActivityTypeEditedWindowsUpdates{
				TeamID:          &team.ID,
				TeamName:        &team.Name,
				DeadlineDays:    deadline,
				GracePeriodDays: grace,
			},
		); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create activity for team macos min version edited")
		}
	}
	if macOSDiskEncryptionUpdated {
		var act fleet.ActivityDetails
		if team.Config.MDM.EnableDiskEncryption {
			act = fleet.ActivityTypeEnabledMacosDiskEncryption{TeamID: &team.ID, TeamName: &team.Name}
			if err := svc.MDMAppleEnableFileVaultAndEscrow(ctx, &team.ID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "enable team filevault and escrow")
			}
		} else {
			act = fleet.ActivityTypeDisabledMacosDiskEncryption{TeamID: &team.ID, TeamName: &team.Name}
			if err := svc.MDMAppleDisableFileVaultAndEscrow(ctx, &team.ID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "disable team filevault and escrow")
			}
		}
		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), act); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create activity for team macos disk encryption")
		}
	}
	if macOSEnableEndUserAuthUpdated {
		if err := svc.updateMacOSSetupEnableEndUserAuth(ctx, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication, &team.ID, &team.Name); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "update macos setup enable end user auth")
		}
	}
	return team, err
}

func (svc *Service) ModifyTeamAgentOptions(ctx context.Context, teamID uint, teamOptions json.RawMessage, applyOptions fleet.ApplySpecOptions) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	team, err := svc.ds.Team(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if teamOptions != nil {
		if err := fleet.ValidateJSONAgentOptions(ctx, svc.ds, teamOptions, true); err != nil {
			err = fleet.NewUserMessageError(err, http.StatusBadRequest)
			if applyOptions.Force && !applyOptions.DryRun {
				level.Info(svc.logger).Log("err", err, "msg", "force-apply team agent options with validation errors")
			}
			if !applyOptions.Force {
				return nil, ctxerr.Wrap(ctx, err, "validate agent options")
			}
		}
	}
	if applyOptions.DryRun {
		return team, nil
	}

	if teamOptions != nil {
		team.Config.AgentOptions = &teamOptions
	} else {
		team.Config.AgentOptions = nil
	}

	tm, err := svc.ds.SaveTeam(ctx, team)
	if err != nil {
		return nil, err
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeEditedAgentOptions{
			Global:   false,
			TeamID:   &team.ID,
			TeamName: &team.Name,
		},
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create edited agent options activity")
	}

	return tm, nil
}

func (svc *Service) AddTeamUsers(ctx context.Context, teamID uint, users []fleet.TeamUser) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	currentUser := authz.UserFromContext(ctx)

	idMap := make(map[uint]fleet.TeamUser)
	for _, user := range users {
		if !fleet.ValidTeamRole(user.Role) {
			return nil, fleet.NewInvalidArgumentError("users", fmt.Sprintf("%s is not a valid role for a team user", user.Role))
		}
		idMap[user.ID] = user
		fullUser, err := svc.ds.UserByID(ctx, user.ID)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "getting full user with id %d", user.ID)
		}
		if fullUser.GlobalRole != nil && currentUser.GlobalRole == nil {
			return nil, ctxerr.New(ctx, "A user with a global role cannot be added to a team by a non global user.")
		}
	}

	team, err := svc.ds.Team(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Replace existing
	for i, existingUser := range team.Users {
		if user, ok := idMap[existingUser.ID]; ok {
			team.Users[i] = user
			delete(idMap, user.ID)
		}
	}

	// Add new (that have not already been replaced)
	for _, user := range idMap {
		team.Users = append(team.Users, user)
	}

	logging.WithExtras(ctx, "users", team.Users)

	return svc.ds.SaveTeam(ctx, team)
}

func (svc *Service) DeleteTeamUsers(ctx context.Context, teamID uint, users []fleet.TeamUser) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	idMap := make(map[uint]bool)
	for _, user := range users {
		idMap[user.ID] = true
	}

	team, err := svc.ds.Team(ctx, teamID)
	if err != nil {
		return nil, err
	}

	newUsers := []fleet.TeamUser{}
	// Delete existing
	for _, existingUser := range team.Users {
		if _, ok := idMap[existingUser.ID]; !ok {
			// Only add non-deleted users
			newUsers = append(newUsers, existingUser)
		}
	}
	team.Users = newUsers

	logging.WithExtras(ctx, "users", team.Users)

	return svc.ds.SaveTeam(ctx, team)
}

func (svc *Service) ListTeamUsers(ctx context.Context, teamID uint, opt fleet.ListOptions) ([]*fleet.User, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	team, err := svc.ds.Team(ctx, teamID)
	if err != nil {
		return nil, err
	}

	return svc.ds.ListUsers(ctx, fleet.UserListOptions{ListOptions: opt, TeamID: team.ID})
}

func (svc *Service) ListTeams(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	teams, err := svc.ds.ListTeams(ctx, filter, opt)
	if err != nil {
		return nil, err
	}

	if err = obfuscateSecrets(vc.User, teams); err != nil {
		return nil, err
	}

	return teams, nil
}

func (svc *Service) ListAvailableTeamsForUser(ctx context.Context, user *fleet.User) ([]*fleet.TeamSummary, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	availableTeams := []*fleet.TeamSummary{}
	if user.GlobalRole != nil {
		ts, err := svc.ds.TeamsSummary(ctx)
		if err != nil {
			return nil, err
		}
		availableTeams = append(availableTeams, ts...)
	} else {
		for _, t := range user.Teams {
			// Convert from UserTeam to TeamSummary (i.e. omit the role, counts, agent options)
			availableTeams = append(availableTeams, &fleet.TeamSummary{ID: t.ID, Name: t.Name, Description: t.Description})
		}
	}

	return availableTeams, nil
}

func (svc *Service) DeleteTeam(ctx context.Context, teamID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	team, err := svc.ds.Team(ctx, teamID)
	if err != nil {
		return err
	}
	name := team.Name

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	opts := fleet.HostListOptions{
		TeamFilter:    &teamID,
		DisableIssues: true, // don't need to check policies for hosts that are being deleted
	}

	hosts, err := svc.ds.ListHosts(ctx, filter, opts)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list hosts for reconcile profiles on team change")
	}
	hostIDs := make([]uint, 0, len(hosts))
	mdmHostSerials := make([]string, 0, len(hosts))
	for _, host := range hosts {
		hostIDs = append(hostIDs, host.ID)
		if host.IsDEPAssignedToFleet() {
			mdmHostSerials = append(mdmHostSerials, host.HardwareSerial)
		}
	}

	if err := svc.ds.DeleteTeam(ctx, teamID); err != nil {
		return err
	}

	if len(hostIDs) > 0 {
		if _, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, hostIDs, nil, nil, nil); err != nil {
			return ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
		}

		if err := svc.ds.CleanupDiskEncryptionKeysOnTeamChange(ctx, hostIDs, ptr.Uint(0)); err != nil {
			return ctxerr.Wrap(ctx, err, "reconcile profiles on team change cleanup disk encryption keys")
		}

		if len(mdmHostSerials) > 0 {
			if _, err := worker.QueueMacosSetupAssistantJob(
				ctx,
				svc.ds,
				svc.logger,
				worker.MacosSetupAssistantTeamDeleted,
				nil,
				mdmHostSerials...); err != nil {
				return ctxerr.Wrap(ctx, err, "queue macos setup assistant team deleted job")
			}
		}
	}

	logging.WithExtras(ctx, "id", teamID)

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedTeam{
			ID:   teamID,
			Name: name,
		},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for team deletion")
	}
	return nil
}

func (svc *Service) GetTeam(ctx context.Context, teamID uint) (*fleet.Team, error) {
	alreadyAuthd := svc.authz.IsAuthenticatedWith(ctx, authz_ctx.AuthnDeviceToken)
	if alreadyAuthd {
		// device-authenticated request can only get the device's team
		host, ok := hostctx.FromContext(ctx)
		if !ok {
			err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
			return nil, err
		}
		if host.TeamID == nil || *host.TeamID != teamID {
			return nil, authz.ForbiddenWithInternal("device-authenticated host does not belong to requested team", nil, "team", "read")
		}
	} else {
		if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionRead); err != nil {
			return nil, err
		}
	}

	logging.WithExtras(ctx, "id", teamID)

	var user *fleet.User
	if alreadyAuthd {
		// device-authenticated, there is no user in the context, use a global
		// observer with no special permissions
		user = &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)}
	} else {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, fleet.ErrNoContext
		}
		user = vc.User
	}

	team, err := svc.ds.Team(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if err = obfuscateSecrets(user, []*fleet.Team{team}); err != nil {
		return nil, err
	}

	return team, nil
}

func (svc *Service) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*fleet.EnrollSecret, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	secrets, err := svc.ds.TeamEnrollSecrets(ctx, teamID)
	if err != nil {
		return nil, err
	}

	isGlobalObs := vc.User.IsGlobalObserver()
	teamMemberships := vc.User.TeamMembership(func(t fleet.UserTeam) bool {
		return true
	})
	obsMembership := vc.User.TeamMembership(func(t fleet.UserTeam) bool {
		return t.Role == fleet.RoleObserver || t.Role == fleet.RoleObserverPlus
	})

	for _, s := range secrets {
		if s == nil {
			continue
		}
		if isGlobalObs || vc.User.GlobalRole == nil && (!teamMemberships[*s.TeamID] || obsMembership[*s.TeamID]) {
			s.Secret = fleet.MaskedPassword
		}
	}

	return secrets, nil
}

func (svc *Service) ModifyTeamEnrollSecrets(ctx context.Context, teamID uint, secrets []fleet.EnrollSecret) ([]*fleet.EnrollSecret, error) {
	if err := svc.authz.Authorize(ctx, &fleet.EnrollSecret{TeamID: ptr.Uint(teamID)}, fleet.ActionWrite); err != nil {
		return nil, err
	}
	if secrets == nil {
		return nil, fleet.NewInvalidArgumentError("secrets", "missing required argument")
	}
	if len(secrets) > fleet.MaxEnrollSecretsCount {
		return nil, fleet.NewInvalidArgumentError("secrets", "too many secrets")
	}

	var newSecrets []*fleet.EnrollSecret
	for _, secret := range secrets {
		newSecrets = append(newSecrets, &fleet.EnrollSecret{
			Secret: secret.Secret,
		})
	}
	if err := svc.ds.ApplyEnrollSecrets(ctx, ptr.Uint(teamID), newSecrets); err != nil {
		return nil, err
	}

	return newSecrets, nil
}

func (svc *Service) teamByIDOrName(ctx context.Context, id *uint, name *string) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	var (
		tm  *fleet.Team
		err error
	)
	if id != nil {
		tm, err = svc.ds.Team(ctx, *id)
		if err != nil {
			return nil, err
		}
	} else if name != nil {
		tm, err = svc.ds.TeamByName(ctx, *name)
		if err != nil {
			return nil, err
		}
	}
	return tm, nil
}

var jsonNull = json.RawMessage(`null`)

// setAuthCheckedOnPreAuthErr can be used to set the authentication as checked
// in case of errors that happened before an auth check can be performed.
// Otherwise the endpoints return a "authentication skipped" error instead of
// the actual returned error.
func setAuthCheckedOnPreAuthErr(ctx context.Context) {
	if az, ok := authz_ctx.FromContext(ctx); ok {
		az.SetChecked()
	}
}

func (svc *Service) checkAuthorizationForTeams(ctx context.Context, specs []*fleet.TeamSpec) error {
	for _, spec := range specs {
		var team *fleet.Team
		var err error
		// If filename is provided, try to find the team by filename first.
		// This is needed in case user is trying to modify the team name.
		if spec.Filename != nil && *spec.Filename != "" {
			team, err = svc.ds.TeamByFilename(ctx, *spec.Filename)
			if err != nil && !fleet.IsNotFound(err) {
				return err
			}
		}
		if team == nil {
			team, err = svc.ds.TeamByName(ctx, spec.Name)
			if err != nil {
				if fleet.IsNotFound(err) {
					// Can the user create a new team?
					if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionWrite); err != nil {
						return err
					}
					continue
				}

				// Set authorization as checked to return a proper error.
				setAuthCheckedOnPreAuthErr(ctx)
				return err
			}
		}

		// can the user modify each team it's trying to modify
		if err := svc.authz.Authorize(ctx, team, fleet.ActionWrite); err != nil {
			return err
		}
	}
	return nil
}

func (svc *Service) ApplyTeamSpecs(ctx context.Context, specs []*fleet.TeamSpec, applyOpts fleet.ApplyTeamSpecOptions) (
	map[string]uint, error,
) {
	if len(specs) == 0 {
		setAuthCheckedOnPreAuthErr(ctx)
		// Nothing to do.
		return map[string]uint{}, nil
	}

	if err := svc.checkAuthorizationForTeams(ctx, specs); err != nil {
		return nil, err
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, err
	}
	appConfig.Obfuscate()

	var details []fleet.TeamActivityDetail

	for _, spec := range specs {
		var secrets []*fleet.EnrollSecret
		// When secrets slice is empty, all secrets are removed.
		// When secrets slice is nil, existing secrets are kept.
		if spec.Secrets != nil {
			secrets = make([]*fleet.EnrollSecret, 0, len(*spec.Secrets))
			for _, secret := range *spec.Secrets {
				secrets = append(
					secrets, &fleet.EnrollSecret{
						Secret: secret.Secret,
					},
				)
			}
		}

		l := strings.ToLower(spec.Name)
		if l == strings.ToLower(fleet.ReservedNameAllTeams) {
			return nil, fleet.NewInvalidArgumentError("name", `"All teams" is a reserved team name`)
		}
		if l == strings.ToLower(fleet.ReservedNameNoTeam) {
			return nil, fleet.NewInvalidArgumentError("name", `"No team" is a reserved team name`)
		}

		var team *fleet.Team
		// If filename is provided, try to find the team by filename first.
		// This is needed in case user is trying to modify the team name.
		if spec.Filename != nil && *spec.Filename != "" {
			team, err = svc.ds.TeamByFilename(ctx, *spec.Filename)
			if err != nil && !fleet.IsNotFound(err) {
				return nil, err
			}
			if team != nil && team.Name != spec.Name {
				// If user is trying to change team name, check that the new name is not already taken.
				_, err = svc.ds.TeamByName(ctx, spec.Name)
				switch {
				case err == nil:
					return nil, fleet.NewInvalidArgumentError("name",
						fmt.Sprintf("cannot change team name from '%s' (filename: %s) to '%s' because team name already exists", team.Name,
							*spec.Filename, spec.Name))
				case fleet.IsNotFound(err):
					// OK
				default:
					return nil, err
				}
			}
		}

		var create bool
		if team == nil {
			team, err = svc.ds.TeamByName(ctx, spec.Name)
			switch {
			case err == nil:
				// OK
			case fleet.IsNotFound(err):
				if spec.Name == "" {
					return nil, fleet.NewInvalidArgumentError("name", "name may not be empty")
				}
				create = true
			default:
				return nil, err
			}
		}

		if len(spec.AgentOptions) > 0 && !bytes.Equal(spec.AgentOptions, jsonNull) {
			if err := fleet.ValidateJSONAgentOptions(ctx, svc.ds, spec.AgentOptions, true); err != nil {
				err = fleet.NewUserMessageError(err, http.StatusBadRequest)
				if applyOpts.Force && !applyOpts.DryRun {
					level.Info(svc.logger).Log("err", err, "msg", "force-apply team agent options with validation errors")
				}
				if !applyOpts.Force {
					return nil, ctxerr.Wrap(ctx, err, "validate agent options")
				}
			}
		}
		if len(secrets) > fleet.MaxEnrollSecretsCount {
			return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("secrets", "too many secrets"), "validate secrets")
		}
		if err := spec.MDM.MacOSUpdates.Validate(); err != nil {
			return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("macos_updates", err.Error()))
		}
		if err := spec.MDM.WindowsUpdates.Validate(); err != nil {
			return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("windows_updates", err.Error()))
		}

		if create {

			// create a new team enroll secret if none is provided for a new team,
			// unless the user explicitly passed in an empty array
			if secrets == nil {
				secret, err := server.GenerateRandomText(fleet.EnrollSecretDefaultLength)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, "generate enroll secret string")
				}
				secrets = append(secrets, &fleet.EnrollSecret{
					Secret: secret,
				})
			}

			team, err := svc.createTeamFromSpec(ctx, spec, appConfig, secrets, applyOpts.DryRun)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "creating team from spec")
			}
			details = append(details, fleet.TeamActivityDetail{
				ID:   team.ID,
				Name: team.Name,
			})
			continue
		}

		if err := svc.editTeamFromSpec(ctx, team, spec, appConfig, secrets, applyOpts); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "editing team from spec")
		}

		details = append(details, fleet.TeamActivityDetail{
			ID:   team.ID,
			Name: team.Name,
		})
	}

	idsByName := make(map[string]uint, len(details))
	if len(details) > 0 {
		for _, tm := range details {
			idsByName[tm.Name] = tm.ID
		}

		if !applyOpts.DryRun {
			if err := svc.NewActivity(
				ctx,
				authz.UserFromContext(ctx),
				fleet.ActivityTypeAppliedSpecTeam{
					Teams: details,
				},
			); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "create activity for team spec")
			}
		}
	}
	return idsByName, nil
}

func (svc *Service) createTeamFromSpec(
	ctx context.Context,
	spec *fleet.TeamSpec,
	appCfg *fleet.AppConfig,
	secrets []*fleet.EnrollSecret,
	dryRun bool,
) (*fleet.Team, error) {
	agentOptions := &spec.AgentOptions
	if len(spec.AgentOptions) == 0 {
		agentOptions = appCfg.AgentOptions
	}

	// if a team spec is not provided, use the global features, otherwise
	// build a new config from the spec with default values applied.
	var err error
	features := appCfg.Features
	if spec.Features != nil {
		features, err = unmarshalWithGlobalDefaults(spec.Features)
		if err != nil {
			return nil, err
		}
	}

	var macOSSettings fleet.MacOSSettings
	if err := svc.applyTeamMacOSSettings(ctx, spec, &macOSSettings); err != nil {
		return nil, err
	}
	macOSSetup := spec.MDM.MacOSSetup
	if !macOSSetup.EnableReleaseDeviceManually.Valid {
		macOSSetup.EnableReleaseDeviceManually = optjson.SetBool(false)
	}
	if macOSSetup.MacOSSetupAssistant.Value != "" || macOSSetup.BootstrapPackage.Value != "" || macOSSetup.EnableReleaseDeviceManually.Value {
		if !appCfg.MDM.EnabledAndConfigured {
			return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("macos_setup",
				`Couldn't update macos_setup because MDM features aren't turned on in Fleet. Use fleetctl generate mdm-apple and then fleet serve with mdm configuration to turn on MDM features.`))
		}
	}

	enableDiskEncryption := spec.MDM.EnableDiskEncryption.Value
	if !spec.MDM.EnableDiskEncryption.Valid {
		if de := macOSSettings.DeprecatedEnableDiskEncryption; de != nil {
			enableDiskEncryption = *de
		}
	}

	invalid := &fleet.InvalidArgumentError{}
	if enableDiskEncryption && svc.config.Server.PrivateKey == "" {
		return nil, ctxerr.New(ctx, "Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
	}
	validateTeamCustomSettings(invalid, "macos", macOSSettings.CustomSettings)
	validateTeamCustomSettings(invalid, "windows", spec.MDM.WindowsSettings.CustomSettings.Value)

	var hostExpirySettings fleet.HostExpirySettings
	if spec.HostExpirySettings != nil {
		if spec.HostExpirySettings.HostExpiryEnabled && spec.HostExpirySettings.HostExpiryWindow <= 0 {
			invalid.Append(
				"host_expiry_settings.host_expiry_window", "When enabling host expiry, host expiry window must be a positive number.",
			)
		}
		hostExpirySettings = *spec.HostExpirySettings
	}

	var hostStatusWebhook *fleet.HostStatusWebhookSettings
	if spec.WebhookSettings.HostStatusWebhook != nil {
		fleet.ValidateEnabledHostStatusIntegrations(*spec.WebhookSettings.HostStatusWebhook, invalid)
		hostStatusWebhook = spec.WebhookSettings.HostStatusWebhook
	}

	if spec.Integrations.GoogleCalendar != nil {
		err = svc.validateTeamCalendarIntegrations(spec.Integrations.GoogleCalendar, appCfg, dryRun, invalid)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "validate team calendar integrations")
		}
	}

	if dryRun {
		for _, secret := range secrets {
			available, err := svc.ds.IsEnrollSecretAvailable(ctx, secret.Secret, true, nil)
			if err != nil {
				return nil, err
			}
			if !available {
				invalid.Append("secrets", fmt.Sprintf("a provided enroll secret for team '%s' is already being used", spec.Name))
				break
			}
		}
	}

	if invalid.HasErrors() {
		return nil, ctxerr.Wrap(ctx, invalid)
	}

	if dryRun {
		return &fleet.Team{Name: spec.Name}, nil
	}

	tm, err := svc.ds.NewTeam(ctx, &fleet.Team{
		Name:     spec.Name,
		Filename: spec.Filename,
		Config: fleet.TeamConfig{
			AgentOptions: agentOptions,
			Features:     features,
			MDM: fleet.TeamMDM{
				EnableDiskEncryption: enableDiskEncryption,
				MacOSUpdates:         spec.MDM.MacOSUpdates,
				WindowsUpdates:       spec.MDM.WindowsUpdates,
				MacOSSettings:        macOSSettings,
				MacOSSetup:           macOSSetup,
				WindowsSettings:      spec.MDM.WindowsSettings,
			},
			HostExpirySettings: hostExpirySettings,
			WebhookSettings: fleet.TeamWebhookSettings{
				HostStatusWebhook: hostStatusWebhook,
			},
			Integrations: fleet.TeamIntegrations{
				GoogleCalendar: spec.Integrations.GoogleCalendar,
			},
			Software: spec.Software,
		},
		Secrets: secrets,
	})
	if err != nil {
		return nil, err
	}

	if enableDiskEncryption && appCfg.MDM.EnabledAndConfigured {
		// TODO: Are we missing an activity or anything else for BitLocker here?
		if err := svc.MDMAppleEnableFileVaultAndEscrow(ctx, &tm.ID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "enable team filevault and escrow")
		}

		if err := svc.NewActivity(
			ctx,
			authz.UserFromContext(ctx),
			fleet.ActivityTypeEnabledMacosDiskEncryption{TeamID: &tm.ID, TeamName: &tm.Name},
		); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create activity for team macos disk encryption")
		}
	}
	return tm, nil
}

func (svc *Service) editTeamFromSpec(
	ctx context.Context,
	team *fleet.Team,
	spec *fleet.TeamSpec,
	appCfg *fleet.AppConfig,
	secrets []*fleet.EnrollSecret,
	opts fleet.ApplyTeamSpecOptions,
) error {
	if !opts.DryRun {
		// We keep the original name for dry run because subsequent dry run calls may need the original name to fetch the team
		team.Name = spec.Name
	}
	team.Filename = spec.Filename

	// if agent options are not provided, do not change them
	if len(spec.AgentOptions) > 0 {
		if bytes.Equal(spec.AgentOptions, jsonNull) {
			// agent options provided but null, clear existing agent option
			team.Config.AgentOptions = nil
		} else {
			team.Config.AgentOptions = &spec.AgentOptions
		}
	}

	// replace (don't merge) the features with the new ones, using a config
	// that has the global defaults applied.
	features, err := unmarshalWithGlobalDefaults(spec.Features)
	if err != nil {
		return err
	}
	team.Config.Features = features
	var mdmMacOSUpdatesEdited bool
	if spec.MDM.MacOSUpdates.Deadline.Set || spec.MDM.MacOSUpdates.MinimumVersion.Set {
		team.Config.MDM.MacOSUpdates = spec.MDM.MacOSUpdates
		mdmMacOSUpdatesEdited = true
	}
	var mdmIOSUpdatesEdited bool
	if spec.MDM.IOSUpdates.Deadline.Set || spec.MDM.IOSUpdates.MinimumVersion.Set {
		team.Config.MDM.IOSUpdates = spec.MDM.IOSUpdates
		mdmIOSUpdatesEdited = true
	}
	var mdmIPadOSUpdatesEdited bool
	if spec.MDM.IPadOSUpdates.Deadline.Set || spec.MDM.IPadOSUpdates.MinimumVersion.Set {
		team.Config.MDM.IPadOSUpdates = spec.MDM.IPadOSUpdates
		mdmIPadOSUpdatesEdited = true
	}
	if spec.MDM.WindowsUpdates.DeadlineDays.Set || spec.MDM.WindowsUpdates.GracePeriodDays.Set {
		team.Config.MDM.WindowsUpdates = spec.MDM.WindowsUpdates
	}

	oldEnableDiskEncryption := team.Config.MDM.EnableDiskEncryption
	if err := svc.applyTeamMacOSSettings(ctx, spec, &team.Config.MDM.MacOSSettings); err != nil {
		return err
	}

	// 1. if the spec has the new setting, use that
	// 2. else if the spec has the deprecated setting, use that
	// 3. otherwise, leave the setting untouched
	if spec.MDM.EnableDiskEncryption.Valid {
		team.Config.MDM.EnableDiskEncryption = spec.MDM.EnableDiskEncryption.Value
	} else if de := team.Config.MDM.MacOSSettings.DeprecatedEnableDiskEncryption; de != nil {
		team.Config.MDM.EnableDiskEncryption = *de
	}
	didUpdateDiskEncryption := team.Config.MDM.EnableDiskEncryption != oldEnableDiskEncryption

	if didUpdateDiskEncryption && team.Config.MDM.EnableDiskEncryption && svc.config.Server.PrivateKey == "" {
		return ctxerr.New(ctx, "Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
	}
	if !team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Valid {
		team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually = optjson.SetBool(false)
	}
	oldMacOSSetup := team.Config.MDM.MacOSSetup
	var didUpdateSetupAssistant, didUpdateBootstrapPackage, didUpdateEnableReleaseManually bool
	if spec.MDM.MacOSSetup.MacOSSetupAssistant.Set {
		didUpdateSetupAssistant = oldMacOSSetup.MacOSSetupAssistant.Value != spec.MDM.MacOSSetup.MacOSSetupAssistant.Value
		team.Config.MDM.MacOSSetup.MacOSSetupAssistant = spec.MDM.MacOSSetup.MacOSSetupAssistant
	}
	if spec.MDM.MacOSSetup.BootstrapPackage.Set {
		didUpdateBootstrapPackage = oldMacOSSetup.BootstrapPackage.Value != spec.MDM.MacOSSetup.BootstrapPackage.Value
		team.Config.MDM.MacOSSetup.BootstrapPackage = spec.MDM.MacOSSetup.BootstrapPackage
	}
	if spec.MDM.MacOSSetup.EnableReleaseDeviceManually.Valid {
		didUpdateEnableReleaseManually = oldMacOSSetup.EnableReleaseDeviceManually.Value != spec.MDM.MacOSSetup.EnableReleaseDeviceManually.Value
		team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually = spec.MDM.MacOSSetup.EnableReleaseDeviceManually
	}
	// TODO(mna): doesn't look like we create an activity for macos updates when
	// modified via spec? Doing the same for Windows, but should we?

	if !appCfg.MDM.EnabledAndConfigured &&
		((didUpdateSetupAssistant && team.Config.MDM.MacOSSetup.MacOSSetupAssistant.Value != "") ||
			(didUpdateBootstrapPackage && team.Config.MDM.MacOSSetup.BootstrapPackage.Value != "") ||
			(didUpdateEnableReleaseManually && team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)) {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("macos_setup",
			`Couldn't update macos_setup because MDM features aren't turned on in Fleet. Use fleetctl generate mdm-apple and then fleet serve with mdm configuration to turn on MDM features.`))
	}

	didUpdateMacOSEndUserAuth := spec.MDM.MacOSSetup.EnableEndUserAuthentication != oldMacOSSetup.EnableEndUserAuthentication
	if didUpdateMacOSEndUserAuth && spec.MDM.MacOSSetup.EnableEndUserAuthentication {
		if !appCfg.MDM.EnabledAndConfigured {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("macos_setup.enable_end_user_authentication",
				`Couldn't update macos_setup.enable_end_user_authentication because MDM features aren't turned on in Fleet. Use fleetctl generate mdm-apple and then fleet serve with mdm configuration to turn on MDM features.`))
		}
		if appCfg.MDM.EndUserAuthentication.IsEmpty() {
			// TODO: update this error message to include steps to resolve the issue once docs for IdP
			// config are available
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("macos_setup.enable_end_user_authentication",
				`Couldn't enable macos_setup.enable_end_user_authentication because no IdP is configured for MDM features.`))
		}
	}
	if didUpdateMacOSEndUserAuth {
		if err := svc.validateEndUserAuthenticationAndSetupAssistant(ctx, &team.ID); err != nil {
			return err
		}
	}
	team.Config.MDM.MacOSSetup.EnableEndUserAuthentication = spec.MDM.MacOSSetup.EnableEndUserAuthentication

	windowsEnabledAndConfigured := appCfg.MDM.WindowsEnabledAndConfigured
	if opts.DryRunAssumptions != nil && opts.DryRunAssumptions.WindowsEnabledAndConfigured.Valid {
		windowsEnabledAndConfigured = opts.DryRunAssumptions.WindowsEnabledAndConfigured.Value
	}
	if spec.MDM.WindowsSettings.CustomSettings.Set {
		if !windowsEnabledAndConfigured &&
			len(spec.MDM.WindowsSettings.CustomSettings.Value) > 0 &&
			!fleet.MDMProfileSpecsMatch(team.Config.MDM.WindowsSettings.CustomSettings.Value, spec.MDM.WindowsSettings.CustomSettings.Value) {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("windows_settings.custom_settings",
				`Couldn’t edit windows_settings.custom_settings. Windows MDM isn’t turned on. Visit https://fleetdm.com/docs/using-fleet to learn how to turn on MDM.`))
		}

		team.Config.MDM.WindowsSettings.CustomSettings = spec.MDM.WindowsSettings.CustomSettings
	}

	if spec.Scripts.Set {
		team.Config.Scripts = spec.Scripts
	}

	if spec.Software != nil {
		if team.Config.Software == nil {
			team.Config.Software = &fleet.SoftwareSpec{}
		}

		if spec.Software.Packages.Set {
			team.Config.Software.Packages = spec.Software.Packages
		}

		if spec.Software.AppStoreApps.Set {
			team.Config.Software.AppStoreApps = spec.Software.AppStoreApps
		}
	}

	if secrets != nil {
		team.Secrets = secrets
	}

	// if host_expiry_settings are not provided, do not change them
	invalid := &fleet.InvalidArgumentError{}
	if spec.HostExpirySettings != nil {
		if spec.HostExpirySettings.HostExpiryEnabled && spec.HostExpirySettings.HostExpiryWindow <= 0 {
			invalid.Append(
				"host_expiry_settings.host_expiry_window", "When enabling host expiry, host expiry window must be a positive number.",
			)
		}
		team.Config.HostExpirySettings = *spec.HostExpirySettings
	}

	validateTeamCustomSettings(invalid, "macos", team.Config.MDM.MacOSSettings.CustomSettings)
	validateTeamCustomSettings(invalid, "windows", team.Config.MDM.WindowsSettings.CustomSettings.Value)

	// If host status webhook is not provided, do not change it
	if spec.WebhookSettings.HostStatusWebhook != nil {
		fleet.ValidateEnabledHostStatusIntegrations(*spec.WebhookSettings.HostStatusWebhook, invalid)
		team.Config.WebhookSettings.HostStatusWebhook = spec.WebhookSettings.HostStatusWebhook
	}

	if spec.Integrations.GoogleCalendar != nil {
		err = svc.validateTeamCalendarIntegrations(spec.Integrations.GoogleCalendar, appCfg, opts.DryRun, invalid)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "validate team calendar integrations")
		}
		team.Config.Integrations.GoogleCalendar = spec.Integrations.GoogleCalendar
	}

	if opts.DryRun {
		for _, secret := range secrets {
			available, err := svc.ds.IsEnrollSecretAvailable(ctx, secret.Secret, false, &team.ID)
			if err != nil {
				return err
			}
			if !available {
				invalid.Append("secrets", fmt.Sprintf("a provided enroll secret for team '%s' is already being used", spec.Name))
				break
			}
		}
	}

	if invalid.HasErrors() {
		return ctxerr.Wrap(ctx, invalid)
	}

	if opts.DryRun {
		return nil
	}

	if _, err := svc.ds.SaveTeam(ctx, team); err != nil {
		return err
	}

	// If no secrets are provided and user did not explicitly specify an empty list, do not replace secrets. (#6774)
	if secrets != nil {
		if err := svc.ds.ApplyEnrollSecrets(ctx, ptr.Uint(team.ID), secrets); err != nil {
			return err
		}
	}
	if appCfg.MDM.EnabledAndConfigured && didUpdateDiskEncryption {
		// TODO: Are we missing an activity or anything else for BitLocker here?
		var act fleet.ActivityDetails
		if team.Config.MDM.EnableDiskEncryption {
			act = fleet.ActivityTypeEnabledMacosDiskEncryption{TeamID: &team.ID, TeamName: &team.Name}
			if err := svc.MDMAppleEnableFileVaultAndEscrow(ctx, &team.ID); err != nil {
				return ctxerr.Wrap(ctx, err, "enable team filevault and escrow")
			}
		} else {
			act = fleet.ActivityTypeDisabledMacosDiskEncryption{TeamID: &team.ID, TeamName: &team.Name}
			if err := svc.MDMAppleDisableFileVaultAndEscrow(ctx, &team.ID); err != nil {
				return ctxerr.Wrap(ctx, err, "disable team filevault and escrow")
			}
		}
		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), act); err != nil {
			return ctxerr.Wrap(ctx, err, "create activity for team macos disk encryption")
		}
	}

	// if the macos setup assistant was cleared, remove it for that team
	if spec.MDM.MacOSSetup.MacOSSetupAssistant.Set &&
		spec.MDM.MacOSSetup.MacOSSetupAssistant.Value == "" &&
		oldMacOSSetup.MacOSSetupAssistant.Value != "" {
		if err := svc.DeleteMDMAppleSetupAssistant(ctx, &team.ID); err != nil {
			return ctxerr.Wrapf(ctx, err, "clear macos setup assistant for team %d", team.ID)
		}
	}

	// if the bootstrap package was cleared, remove it for that team
	if spec.MDM.MacOSSetup.BootstrapPackage.Set &&
		spec.MDM.MacOSSetup.BootstrapPackage.Value == "" &&
		oldMacOSSetup.BootstrapPackage.Value != "" {
		if err := svc.DeleteMDMAppleBootstrapPackage(ctx, &team.ID); err != nil {
			return ctxerr.Wrapf(ctx, err, "clear bootstrap package for team %d", team.ID)
		}
	}

	// if the setup experience script was cleared, remove it for that team
	if spec.MDM.MacOSSetup.Script.Set &&
		spec.MDM.MacOSSetup.Script.Value == "" &&
		oldMacOSSetup.Script.Value != "" {
		if err := svc.DeleteSetupExperienceScript(ctx, &team.ID); err != nil {
			return ctxerr.Wrapf(ctx, err, "clear setup experience script for team %d", team.ID)
		}
	}

	if didUpdateMacOSEndUserAuth {
		if err := svc.updateMacOSSetupEnableEndUserAuth(
			ctx, spec.MDM.MacOSSetup.EnableEndUserAuthentication, &team.ID, &team.Name,
		); err != nil {
			return err
		}
	}

	if mdmMacOSUpdatesEdited {
		if err := svc.mdmAppleEditedAppleOSUpdates(ctx, &team.ID, fleet.MacOS, team.Config.MDM.MacOSUpdates); err != nil {
			return err
		}
	}
	if mdmIOSUpdatesEdited {
		if err := svc.mdmAppleEditedAppleOSUpdates(ctx, &team.ID, fleet.IOS, team.Config.MDM.IOSUpdates); err != nil {
			return err
		}
	}
	if mdmIPadOSUpdatesEdited {
		if err := svc.mdmAppleEditedAppleOSUpdates(ctx, &team.ID, fleet.IPadOS, team.Config.MDM.IPadOSUpdates); err != nil {
			return err
		}
	}

	return nil
}

func validateTeamCustomSettings(invalid *fleet.InvalidArgumentError, prefix string, customSettings []fleet.MDMProfileSpec) {
	for i, prof := range customSettings {
		count := 0
		for _, b := range []bool{len(prof.Labels) > 0, len(prof.LabelsIncludeAll) > 0, len(prof.LabelsIncludeAny) > 0, len(prof.LabelsExcludeAny) > 0} {
			if b {
				count++
			}
		}
		if count > 1 {
			invalid.Append(fmt.Sprintf("%s_settings.custom_settings", prefix),
				fmt.Sprintf(`Couldn't edit %s_settings.custom_settings. For each profile, only one of "labels_exclude_any", "labels_include_all", "labels_include_any" or "labels" can be included.`, prefix))
		}
		if len(prof.Labels) > 0 {
			customSettings[i].LabelsIncludeAll = customSettings[i].Labels
			customSettings[i].Labels = nil
		}
	}
}

func (svc *Service) validateTeamCalendarIntegrations(
	calendarIntegration *fleet.TeamGoogleCalendarIntegration,
	appCfg *fleet.AppConfig, dryRun bool, invalid *fleet.InvalidArgumentError,
) error {
	if !calendarIntegration.Enable {
		return nil
	}
	// Check that global configs exist. During dry run, the global config may not be available yet.
	if len(appCfg.Integrations.GoogleCalendar) == 0 && !dryRun {
		invalid.Append("integrations.google_calendar.enable_calendar_events", "global Google Calendar integration is not configured")
	}
	// Validate URL
	if u, err := url.ParseRequestURI(calendarIntegration.WebhookURL); err != nil {
		invalid.Append("integrations.google_calendar.webhook_url", err.Error())
	} else if u.Scheme != "https" && u.Scheme != "http" {
		invalid.Append("integrations.google_calendar.webhook_url", "webhook_url must be https or http")
	}
	return nil
}

func (svc *Service) applyTeamMacOSSettings(ctx context.Context, spec *fleet.TeamSpec, applyUpon *fleet.MacOSSettings) error {
	oldCustomSettings := applyUpon.CustomSettings
	setFields, err := applyUpon.FromMap(spec.MDM.MacOSSettings)
	if err != nil {
		return fleet.NewUserMessageError(err, http.StatusBadRequest)
	}

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "apply team macos settings")
	}

	customSettingsChanged := setFields["custom_settings"] &&
		len(applyUpon.CustomSettings) > 0 &&
		!fleet.MDMProfileSpecsMatch(applyUpon.CustomSettings, oldCustomSettings)

	if customSettingsChanged || (setFields["enable_disk_encryption"] && *applyUpon.DeprecatedEnableDiskEncryption) {
		field := "custom_settings"
		if !setFields["custom_settings"] {
			field = "enable_disk_encryption"
		}
		if !appCfg.MDM.EnabledAndConfigured {
			// TODO: Address potential edge cases when teams that previously utilized MDM features
			// are edited later edited when MDM disabled
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError(fmt.Sprintf("macos_settings.%s", field),
				`Couldn't update macos_settings because MDM features aren't turned on in Fleet. Use fleetctl generate mdm-apple and then fleet serve with mdm configuration to turn on MDM features.`))
		}
	}

	return nil
}

// unmarshalWithGlobalDefaults unmarshals features from a team spec, and
// assigns default values based on the global defaults for missing fields
func unmarshalWithGlobalDefaults(b *json.RawMessage) (fleet.Features, error) {
	// build a default config with default values applied
	defaults := &fleet.Features{}
	defaults.ApplyDefaultsForNewInstalls()

	// unmarshal the features from the spec into the defaults
	if b != nil {
		if err := json.Unmarshal(*b, defaults); err != nil {
			return fleet.Features{}, err
		}
	}

	return *defaults, nil
}

func (svc *Service) updateTeamMDMDiskEncryption(ctx context.Context, tm *fleet.Team, enable *bool) error {
	var didUpdate, didUpdateMacOSDiskEncryption bool
	if enable != nil {
		if svc.config.Server.PrivateKey == "" {
			return ctxerr.New(ctx, "Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
		}
		if tm.Config.MDM.EnableDiskEncryption != *enable {
			tm.Config.MDM.EnableDiskEncryption = *enable
			didUpdate = true
			didUpdateMacOSDiskEncryption = true
		}
	}

	if didUpdate {
		if _, err := svc.ds.SaveTeam(ctx, tm); err != nil {
			return err
		}

		appCfg, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return err
		}

		// macOS-specific stuff. For legacy reasons we check if apple is configured
		// via `appCfg.MDM.EnabledAndConfigured`
		//
		// TODO: is there a missing bitlocker activity feed item? (see same TODO on
		// other methods that deal with disk encryption)
		if appCfg.MDM.EnabledAndConfigured && didUpdateMacOSDiskEncryption {
			var act fleet.ActivityDetails
			if tm.Config.MDM.EnableDiskEncryption {
				act = fleet.ActivityTypeEnabledMacosDiskEncryption{TeamID: &tm.ID, TeamName: &tm.Name}
				if err := svc.MDMAppleEnableFileVaultAndEscrow(ctx, &tm.ID); err != nil {
					return ctxerr.Wrap(ctx, err, "enable team filevault and escrow")
				}
			} else {
				act = fleet.ActivityTypeDisabledMacosDiskEncryption{TeamID: &tm.ID, TeamName: &tm.Name}
				if err := svc.MDMAppleDisableFileVaultAndEscrow(ctx, &tm.ID); err != nil {
					return ctxerr.Wrap(ctx, err, "disable team filevault and escrow")
				}
			}
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), act); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for team macos disk encryption")
			}
		}
	}
	return nil
}

func (svc *Service) updateTeamMDMAppleSetup(ctx context.Context, tm *fleet.Team, payload fleet.MDMAppleSetupPayload) error {
	var didUpdate, didUpdateMacOSEndUserAuth bool
	if payload.EnableEndUserAuthentication != nil {
		if tm.Config.MDM.MacOSSetup.EnableEndUserAuthentication != *payload.EnableEndUserAuthentication {
			tm.Config.MDM.MacOSSetup.EnableEndUserAuthentication = *payload.EnableEndUserAuthentication
			didUpdate = true
			didUpdateMacOSEndUserAuth = true
		}
	}

	if payload.EnableReleaseDeviceManually != nil {
		if tm.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value != *payload.EnableReleaseDeviceManually {
			tm.Config.MDM.MacOSSetup.EnableReleaseDeviceManually = optjson.SetBool(*payload.EnableReleaseDeviceManually)
			didUpdate = true
		}
	}

	if didUpdate {
		if _, err := svc.ds.SaveTeam(ctx, tm); err != nil {
			return err
		}
		if didUpdateMacOSEndUserAuth {
			if err := svc.updateMacOSSetupEnableEndUserAuth(ctx, tm.Config.MDM.MacOSSetup.EnableEndUserAuthentication, &tm.ID, &tm.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (svc *Service) validateEndUserAuthenticationAndSetupAssistant(ctx context.Context, tmID *uint) error {
	hasCustomConfigurationWebURL, err := svc.HasCustomSetupAssistantConfigurationWebURL(ctx, tmID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking setup assistant configuration web url")
	}

	if hasCustomConfigurationWebURL {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("macos_setup.enable_end_user_authentication", fleet.EndUserAuthDEPWebURLConfiguredErrMsg))
	}

	return nil
}
