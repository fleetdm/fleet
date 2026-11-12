package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ConfigListsByTeamIDs(ctx context.Context, teamIDs []uint) (map[uint]*fleet.ConfigLists, error) {
	// skipauth: The lists are part of a config or fleet response the endpoint
	// already authorized. Each list is then hidden unless the user can read it
	// on its own endpoint, rather than requiring read access to the fleet,
	// which GitOps users don't have.
	svc.authz.SkipAuthorization(ctx)

	lists, err := svc.ds.ConfigListsByTeamIDs(ctx, teamIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get config lists")
	}

	// A user has either a global role or fleet roles, and the global rules
	// don't depend on the fleet, so global users need one check for all fleets.
	vc, _ := viewer.FromContext(ctx)
	globalUser := vc.User != nil && vc.User.GlobalRole != nil
	var anyFleet *fleet.ConfigListsHidden
	for id, l := range lists {
		if globalUser && id != 0 && anyFleet != nil {
			l.Hidden = *anyFleet
			continue
		}
		l.Hidden = svc.hiddenConfigLists(ctx, id)
		if globalUser && id != 0 {
			anyFleet = &l.Hidden
		}
	}
	return lists, nil
}

func (svc *Service) hiddenConfigLists(ctx context.Context, teamID uint) fleet.ConfigListsHidden {
	var tmID *uint
	if teamID != 0 {
		tmID = &teamID
	}
	cannotRead := func(obj any) bool {
		return svc.authz.Authorize(ctx, obj, fleet.ActionRead) != nil
	}
	return fleet.ConfigListsHidden{
		Profiles:         cannotRead(&fleet.MDMConfigProfileAuthz{TeamID: tmID}),
		Scripts:          cannotRead(&fleet.Script{TeamID: tmID}),
		BootstrapPackage: cannotRead(&fleet.MDMAppleBootstrapPackage{TeamID: teamID}),
		SetupAssistant:   cannotRead(&fleet.MDMAppleSetupAssistant{TeamID: tmID}),
		// Same check as GetVPPTokens for users who can read every token.
		VolumePurchasingProgram: teamID != 0 || cannotRead(&fleet.VPPApp{}),
	}
}

func addAppConfigLists(ctx context.Context, svc fleet.Service, appConfig *fleet.AppConfig) error {
	lists, err := svc.ConfigListsByTeamIDs(ctx, []uint{0})
	if err != nil {
		return err
	}
	lists[0].ApplyToAppConfig(appConfig)
	return nil
}

func addTeamConfigLists(ctx context.Context, svc fleet.Service, teams ...*fleet.Team) error {
	teamIDs := make([]uint, 0, len(teams))
	for _, t := range teams {
		teamIDs = append(teamIDs, t.ID)
	}
	lists, err := svc.ConfigListsByTeamIDs(ctx, teamIDs)
	if err != nil {
		return err
	}
	for _, t := range teams {
		lists[t.ID].ApplyToTeamConfig(&t.Config)
	}
	return nil
}
