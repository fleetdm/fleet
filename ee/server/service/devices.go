package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server"
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

func (svc *Service) TriggerMigrateMDMDevice(ctx context.Context, host *fleet.Host) error {
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	if !ac.MDM.EnabledAndConfigured {
		return fleet.ErrMDMNotConfigured
	}

	var bre fleet.BadRequestError
	switch {
	case !ac.MDM.MacOSMigration.Enable:
		bre.InternalErr = ctxerr.New(ctx, "macOS migration not enabled")
	case ac.MDM.MacOSMigration.WebhookURL == "":
		bre.InternalErr = ctxerr.New(ctx, "macOS migration webhook URL not configured")
	case !host.IsOsqueryEnrolled(), !host.MDMInfo.IsDEPCapable(), !host.MDMInfo.IsEnrolledInThirdPartyMDM():
		bre.InternalErr = ctxerr.New(ctx, "host not eligible for macOS migration")
	}
	// TODO: add case to check if webhok has already been sent (if host refetchUntil is not zero?)

	if bre.InternalErr != nil {
		return &bre
	}

	p := fleet.MigrateMDMDeviceWebhookPayload{}
	p.Timestamp = time.Now().UTC()
	p.Host.ID = host.ID
	p.Host.UUID = host.UUID
	p.Host.HardwareSerial = host.HardwareSerial

	if err := server.PostJSONWithTimeout(ctx, ac.MDM.MacOSMigration.WebhookURL, p); err != nil {
		return ctxerr.Wrap(ctx, err, "posting macOS migration webhook")
	}

	return nil
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
