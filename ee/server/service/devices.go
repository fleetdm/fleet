package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ListDevicePolicies(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
	return svc.ds.ListPoliciesForHost(ctx, host)
}

func (svc *Service) RequestEncryptionKeyRotation(ctx context.Context, hostID uint) error {
	return svc.ds.SetDiskEncryptionResetStatus(ctx, hostID, true)
}

func (svc *Service) GetFleetDesktopSummary(ctx context.Context) (fleet.DesktopSummary, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	var sum fleet.DesktopSummary

	host, ok := hostctx.FromContext(ctx)

	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return sum, err
	}

	r, err := svc.ds.FailingPoliciesCount(ctx, host)
	if err != nil {
		return sum, ctxerr.Wrap(ctx, err, "retrieving failing policies")
	}
	sum.FailingPolicies = &r

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return sum, ctxerr.Wrap(ctx, err, "retrieving app config")
	}

	if appCfg.MDM.EnabledAndConfigured &&
		appCfg.MDM.MacOSMigration.Enable &&
		host.IsOsqueryEnrolled() &&
		host.MDMInfo.IsDEPCapable() &&
		host.MDMInfo.IsEnrolledInThirdPartyMDM() {
		sum.Notifications.NeedsMDMMigration = true
	}

	return sum, nil
}
