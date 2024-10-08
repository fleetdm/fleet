package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func (svc *Service) GetOrbitSetupExperienceStatus(ctx context.Context, orbitNodeKey string) (*fleet.SetupExperienceStatusPayload, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)
	host, err := svc.ds.LoadHostByOrbitNodeKey(ctx, orbitNodeKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading host by orbit node key")
	}

	// get the status of the bootstrap package deployment
	bootstrapPkg, err := svc.ds.GetHostMDMMacOSSetup(ctx, host.ID)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "get bootstrap package status")
	}
	// NOTE: bootstrapPkg can be nil if there was none to install.

	// TODO(mna): list MDM commands for that host, check that all are in a final
	// state (I believe it's safe to list all commands as any old ones from
	// before the enrollment have been cleared on re-enroll).
	adminTeamFilter := fleet.TeamFilter{
		User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
	}

	// InstallEnterpriseApplication covers the commands to install fleetd and the
	// bootstrap package.
	installCmds, err := svc.ds.ListMDMCommands(ctx, adminTeamFilter, &fleet.MDMCommandListOptions{
		Filters: fleet.MDMCommandFilters{
			HostIdentifier: host.UUID,
			RequestType:    "InstallEnterpriseApplication",
		},
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list InstallEnterpriseApplication commands")
	}
	// AccountConfiguration covers the (optional) command to setup SSO.
	acctCmds, err := svc.ds.ListMDMCommands(ctx, adminTeamFilter, &fleet.MDMCommandListOptions{
		Filters: fleet.MDMCommandFilters{
			HostIdentifier: host.UUID,
			RequestType:    "AccountConfiguration",
		},
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list AccountConfiguration commands")
	}

	// TODO(mna): we keep track of the bootstrap package status in a specific
	// table, and fleetd is necessarily installed when this endpoint gets called,
	// so the only command to check is the optional AccountConfiguration one (for
	// SSO).
	cmdsCompleted := true
	for _, cmd := range append(installCmds, acctCmds...) {
		// succeeded or failed, it is done (final state)
		if cmd.Status != fleet.MDMAppleStatusAcknowledged && cmd.Status != fleet.MDMAppleStatusError &&
			cmd.Status != fleet.MDMAppleStatusCommandFormatError {
			cmdsCompleted = false
		}
	}

	// TODO(mna): list profiles for that host, check that all are in a final state.
	// TODO(mna): how to unblock the caller after 15m waiting for profiles?
	// TODO(mna): once all software/script are in final state, release device if it's not manually released.

	res, err := svc.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing setup experience results")
	}

	payload := &fleet.SetupExperienceStatusPayload{Software: make([]*fleet.SetupExperienceStatusResult, 0)}
	for _, r := range res {
		if r.IsForScript() {
			payload.Script = r
		}

		if r.IsForSoftware() {
			payload.Software = append(payload.Software, r)
		}
	}

	return payload, nil
}
