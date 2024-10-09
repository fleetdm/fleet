package service

import (
	"context"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
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
	var bootstrapPkgResult *fleet.SetupExperienceBootstrapPackageResult
	if bootstrapPkg != nil {
		bootstrapPkgResult = &fleet.SetupExperienceBootstrapPackageResult{
			Name:   bootstrapPkg.BootstrapPackageName,
			Status: bootstrapPkg.BootstrapPackageStatus,
		}
	}

	// get the status of the configuration profiles
	cfgProfs, err := svc.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get configuration profiles status")
	}
	var cfgProfResults []*fleet.SetupExperienceConfigurationProfileResult
	for _, prof := range cfgProfs {
		// NOTE: DDM profiles (declarations) are ignored because while a device is
		// awaiting to be released, it cannot process a DDM session (at least
		// that's what we noticed during testing).
		if strings.HasPrefix(prof.ProfileUUID, fleet.MDMAppleDeclarationUUIDPrefix) {
			continue
		}

		status := fleet.MDMDeliveryPending
		if prof.Status != nil {
			status = *prof.Status
		}
		cfgProfResults = append(cfgProfResults, &fleet.SetupExperienceConfigurationProfileResult{
			ProfileUUID: prof.ProfileUUID,
			Name:        prof.Name,
			Status:      status,
		})
	}

	// AccountConfiguration covers the (optional) command to setup SSO.
	adminTeamFilter := fleet.TeamFilter{
		User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
	}
	acctCmds, err := svc.ds.ListMDMCommands(ctx, adminTeamFilter, &fleet.MDMCommandListOptions{
		Filters: fleet.MDMCommandFilters{
			HostIdentifier: host.UUID,
			RequestType:    "AccountConfiguration",
		},
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list AccountConfiguration commands")
	}

	var acctCfgResult *fleet.SetupExperienceAccountConfigurationResult
	if len(acctCmds) > 0 {
		// there may be more than one if e.g. the worker job that sends them had to
		// retry, but they would all be processed anyway so we can only care about
		// the first one.
		acctCfgResult = &fleet.SetupExperienceAccountConfigurationResult{
			CommandUUID: acctCmds[0].CommandUUID,
			Status:      acctCmds[0].Status,
		}
	}

	// TODO(mna): how to unblock the caller after 15m waiting for profiles?
	// TODO(mna): once all software/script are in final state, release device if it's not manually released.

	res, err := svc.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing setup experience results")
	}

	payload := &fleet.SetupExperienceStatusPayload{
		BootstrapPackage:      bootstrapPkgResult,
		ConfigurationProfiles: cfgProfResults,
		AccountConfiguration:  acctCfgResult,
		Software:              make([]*fleet.SetupExperienceStatusResult, 0),
	}
	for _, r := range res {
		if r.IsForScript() {
			payload.Script = r
		}

		if r.IsForSoftware() {
			payload.Software = append(payload.Software, r)
		}
	}

	if /* forceRelease || */ isDeviceReadyForRelease(payload) {
		level.Info(svc.logger).Log("msg", "releasing device, all DEP enrollment commands and profiles have completed", "host_uuid", host.UUID)
		if err := svc.mdmAppleCommander.DeviceConfigured(ctx, host.UUID, uuid.NewString()); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "failed to enqueue DeviceConfigured command")
		}
	}

	return payload, nil
}

func isDeviceReadyForRelease(payload *fleet.SetupExperienceStatusPayload) bool {
}
