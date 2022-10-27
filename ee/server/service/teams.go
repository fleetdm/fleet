package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/log/level"
)

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

	team, err = svc.ds.NewTeam(ctx, team)
	if err != nil {
		return nil, err
	}

	if err := svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeCreatedTeam,
		&map[string]interface{}{"team_id": team.ID, "team_name": team.Name},
	); err != nil {
		return nil, err
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
		team.Name = *payload.Name
	}
	if payload.Description != nil {
		team.Description = *payload.Description
	}

	if payload.WebhookSettings != nil {
		team.Config.WebhookSettings = *payload.WebhookSettings
	}

	if payload.Integrations != nil {
		// the team integrations must reference an existing global config integration.
		appCfg, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return nil, err
		}
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

	return svc.ds.SaveTeam(ctx, team)
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
		if err := fleet.ValidateJSONAgentOptions(teamOptions); err != nil {
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

	if err := svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeEditedAgentOptions,
		&map[string]interface{}{"global": false, "team_id": team.ID, "team_name": team.Name},
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

	return svc.ds.ListTeams(ctx, filter, opt)
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

	if err := svc.ds.DeleteTeam(ctx, teamID); err != nil {
		return err
	}

	logging.WithExtras(ctx, "id", teamID)

	return svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedTeam,
		&map[string]interface{}{"team_id": teamID, "team_name": name},
	)
}

func (svc *Service) GetTeam(ctx context.Context, teamID uint) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	logging.WithExtras(ctx, "id", teamID)

	return svc.ds.Team(ctx, teamID)
}

func (svc *Service) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*fleet.EnrollSecret, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.TeamEnrollSecrets(ctx, teamID)
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

func (svc *Service) ApplyTeamSpecs(ctx context.Context, specs []*fleet.TeamSpec, applyOpts fleet.ApplySpecOptions) error {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return err
	}

	// check auth for all teams specified first
	for _, spec := range specs {
		team, err := svc.ds.TeamByName(ctx, spec.Name)
		if err != nil {
			if err := ctxerr.Cause(err); err == sql.ErrNoRows {
				// can the user create a new team?
				if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionWrite); err != nil {
					return err
				}
				continue
			}

			return err
		}

		// can the user modify each team it's trying to modify
		if err := svc.authz.Authorize(ctx, team, fleet.ActionWrite); err != nil {
			return err
		}
	}

	appConfig, err := svc.AppConfig(ctx)
	if err != nil {
		return err
	}

	type activityDetail struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	var details []activityDetail

	for _, spec := range specs {
		var secrets []*fleet.EnrollSecret
		for _, secret := range spec.Secrets {
			secrets = append(secrets, &fleet.EnrollSecret{
				Secret: secret.Secret,
			})
		}

		var create bool
		team, err := svc.ds.TeamByName(ctx, spec.Name)
		switch {
		case err == nil:
			// OK
		case ctxerr.Cause(err) == sql.ErrNoRows:
			if spec.Name == "" {
				return fleet.NewInvalidArgumentError("name", "name may not be empty")
			}
			create = true
		default:
			return err
		}

		if spec.AgentOptions != nil {
			if err := fleet.ValidateJSONAgentOptions(*spec.AgentOptions); err != nil {
				err = fleet.NewUserMessageError(err, http.StatusBadRequest)
				if applyOpts.Force && !applyOpts.DryRun {
					level.Info(svc.logger).Log("err", err, "msg", "force-apply team agent options with validation errors")
				}
				if !applyOpts.Force {
					return ctxerr.Wrap(ctx, err, "validate agent options")
				}
			}
		}
		if len(spec.Secrets) > fleet.MaxEnrollSecretsCount {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("secrets", "too many secrets"), "validate secrets")
		}

		if applyOpts.DryRun {
			continue
		}

		if create {
			team, err := svc.createTeamFromSpec(ctx, spec, appConfig, secrets)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "creating team from spec")
			}
			details = append(details, activityDetail{
				ID:   team.ID,
				Name: team.Name,
			})
			continue
		}

		if err := svc.editTeamFromSpec(ctx, team, spec, secrets); err != nil {
			return ctxerr.Wrap(ctx, err, "editing team from spec")
		}

		details = append(details, activityDetail{
			ID:   team.ID,
			Name: team.Name,
		})
	}

	if len(details) > 0 {
		if err := svc.ds.NewActivity(
			ctx,
			authz.UserFromContext(ctx),
			fleet.ActivityTypeAppliedSpecTeam,
			&map[string]interface{}{"teams": details},
		); err != nil {
			return ctxerr.Wrap(ctx, err, "create applied team spec activity")
		}
	}
	return nil
}

func (svc Service) createTeamFromSpec(ctx context.Context, spec *fleet.TeamSpec, defaults *fleet.AppConfig, secrets []*fleet.EnrollSecret) (*fleet.Team, error) {
	agentOptions := spec.AgentOptions
	if agentOptions == nil {
		agentOptions = defaults.AgentOptions
	}

	// if a team spec is not provided, use the global features, otherwise
	// build a new config from the spec with default values applied.
	var err error
	features := defaults.Features
	if spec.Features != nil {
		features, err = unmarshalWithGlobalDefaults(spec.Features)
		if err != nil {
			return nil, err
		}
	}

	return svc.ds.NewTeam(ctx, &fleet.Team{
		Name: spec.Name,
		Config: fleet.TeamConfig{
			AgentOptions: agentOptions,
			Features:     features,
		},
		Secrets: secrets,
	})
}

func (svc Service) editTeamFromSpec(ctx context.Context, team *fleet.Team, spec *fleet.TeamSpec, secrets []*fleet.EnrollSecret) error {
	team.Name = spec.Name
	team.Config.AgentOptions = spec.AgentOptions

	// replace (don't merge) the features with the new ones, using a config
	// that has the global defaults applied.
	features, err := unmarshalWithGlobalDefaults(spec.Features)
	if err != nil {
		return err
	}
	team.Config.Features = features

	if len(secrets) > 0 {
		team.Secrets = secrets
	}

	if _, err := svc.ds.SaveTeam(ctx, team); err != nil {
		return err
	}

	// only replace enroll secrets if at least one is provided (#6774)
	if len(secrets) > 0 {
		if err := svc.ds.ApplyEnrollSecrets(ctx, ptr.Uint(team.ID), secrets); err != nil {
			return err
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
