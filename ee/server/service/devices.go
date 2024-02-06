package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"
)

func (svc *Service) ListDevicePolicies(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
	return svc.ds.ListPoliciesForHost(ctx, host)
}

func (svc *Service) RequestEncryptionKeyRotation(ctx context.Context, hostID uint) error {
	return svc.ds.SetDiskEncryptionResetStatus(ctx, hostID, true)
}

const refetchMDMUnenrollCriticalQueryDuration = 3 * time.Minute

// TriggerMigrateMDMDevice triggers the webhook associated with the MDM
// migration to Fleet configuration. It is located in the ee package instead of
// the server/webhooks one because it is a Fleet Premium only feature and for
// licensing reasons this needs to live under this package.
func (svc *Service) TriggerMigrateMDMDevice(ctx context.Context, host *fleet.Host) error {
	level.Debug(svc.logger).Log("msg", "trigger migration webhook", "host_id", host.ID,
		"refetch_critical_queries_until", host.RefetchCriticalQueriesUntil)

	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	if !ac.MDM.EnabledAndConfigured {
		return fleet.ErrMDMNotConfigured
	}

	if host.RefetchCriticalQueriesUntil != nil && host.RefetchCriticalQueriesUntil.After(svc.clock.Now()) {
		// the webhook has already been triggered successfully recently (within the
		// refetch critical queries delay), so return as if it did send it successfully
		// but do not re-send.
		level.Debug(svc.logger).Log("msg", "waiting for critical queries refetch, skip sending webhook",
			"host_id", host.ID)
		return nil
	}

	var bre fleet.BadRequestError
	switch {
	case !ac.MDM.MacOSMigration.Enable:
		bre.InternalErr = ctxerr.New(ctx, "macOS migration not enabled")
	case ac.MDM.MacOSMigration.WebhookURL == "":
		bre.InternalErr = ctxerr.New(ctx, "macOS migration webhook URL not configured")
	case !host.IsEligibleForDEPMigration():
		bre.InternalErr = ctxerr.New(ctx, "host not eligible for macOS migration")
	}
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

	// if the webhook was successfully triggered, we update the host to
	// constantly run the query to check if it has been unenrolled from its
	// existing third-party MDM.
	refetchUntil := svc.clock.Now().Add(refetchMDMUnenrollCriticalQueryDuration)
	host.RefetchCriticalQueriesUntil = &refetchUntil
	if err := svc.ds.UpdateHostRefetchCriticalQueriesUntil(ctx, host.ID, &refetchUntil); err != nil {
		return ctxerr.Wrap(ctx, err, "save host with refetch critical queries timestamp")
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

	appCfg, err := svc.AppConfigObfuscated(ctx)
	if err != nil {
		return sum, ctxerr.Wrap(ctx, err, "retrieving app config")
	}

	if appCfg.MDM.EnabledAndConfigured && appCfg.MDM.MacOSMigration.Enable {
		if host.NeedsDEPEnrollment() {
			sum.Notifications.RenewEnrollmentProfile = true
		}

		if host.IsEligibleForDEPMigration() {
			sum.Notifications.NeedsMDMMigration = true
		}
	}

	// organization information
	sum.Config.OrgInfo.OrgName = appCfg.OrgInfo.OrgName
	sum.Config.OrgInfo.OrgLogoURL = appCfg.OrgInfo.OrgLogoURL
	sum.Config.OrgInfo.OrgLogoURLLightBackground = appCfg.OrgInfo.OrgLogoURLLightBackground
	sum.Config.OrgInfo.ContactURL = appCfg.OrgInfo.ContactURL

	// mdm information
	sum.Config.MDM.MacOSMigration.Mode = appCfg.MDM.MacOSMigration.Mode

	return sum, nil
}
