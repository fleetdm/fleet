package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"google.golang.org/api/androidmanagement/v1"
)

func ReconcileProfiles(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, licenseKey string) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get app config")
	}
	if !appConfig.MDM.AndroidEnabledAndConfigured {
		return nil
	}

	// TODO(ap): here would come the queries to identify the profiles to add and
	// remove from the host, and merge the final payload. This will all be part
	// of the upcoming https://github.com/fleetdm/fleet/issues/32032 work, not of
	// the current work. For the current ticket, I'll just assume we have the
	// final payload.
	//
	// Would probably be a good idea to generate the canonical JSON form of the
	// payload and keep track of the hash of the last applied payload, to avoid
	// re-applying if there are no changes. Also, I'm not sure how _removing_ a
	// setting/profile would work, does it get "removed" just by the fact that
	// the settings are not present in the new profile applied?
	//
	// We also need to agree on a determined order to merge the profiles. I'd go
	// by name, alphabetically ascending, as it's simple and the order
	// information can be viewed by the user in the UI, but we had discussed
	// upload time of the profile (which may not be deterministic for batch-set
	// profiles).
	//
	// Due to the logic needed to merge the "profiles" to form a final "policy"
	// payload, I don't think we can use SQL queries to find out what hosts need
	// to be updated or not, I think that at best we can generate a minimal
	// subset of affected hosts via queries by using things like last policy
	// timestamp vs timestamps of the profiles involved, and if it looks like a
	// host may need an update, compute the final payload and use the checksum to
	// see if it has actually changed or not.

	client := newAMAPIClient(ctx, logger, licenseKey)

	// TODO(ap): let's use a simulated policy (that would be generated from the merged profiles)
	// for now.
	basePolicy := &androidmanagement.Policy{
		CameraDisabled: true,
	}

	// TODO(ap): at this point, we'd have a bunch of hosts that need to have their policy
	// updated. Let's simulate it for any existing Android hosts for now.
	filter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
	hosts, err := ds.ListHosts(ctx, filter, fleet.HostListOptions{OSNameFilter: ptr.String("android")})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list android hosts")
	}

	for _, h := range hosts {
		// for every policy, we want to enforce some settings
		applyFleetEnforcedSettings(basePolicy)
		//policyName := fmt.Sprintf("%s/policies/%s", enterprise.Name(), fmt.Sprintf("%d", defaultAndroidPolicyID))
		client.EnterprisesPoliciesPatch(ctx)
	}

	panic("unimplemented")
}

func applyFleetEnforcedSettings(policy *androidmanagement.Policy) {
	// TODO(ap): exact settings to be confirmed, for now using the same settings we
	// use in the (existing) default Android profile.
	policy.StatusReportingSettings = &androidmanagement.StatusReportingSettings{
		DeviceSettingsEnabled:        true,
		MemoryInfoEnabled:            true,
		NetworkInfoEnabled:           true,
		DisplayInfoEnabled:           true,
		PowerManagementEventsEnabled: true,
		HardwareStatusEnabled:        true,
		SystemPropertiesEnabled:      true,
		SoftwareInfoEnabled:          true,
		CommonCriteriaModeEnabled:    true,
		ApplicationReportsEnabled:    false,
		ApplicationReportingSettings: nil,
	}
}
