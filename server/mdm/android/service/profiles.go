package service

import (
	"context"
	"fmt"

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

	// get the one-and-only Android enterprise, which is treated as an error if
	// not present, since the appconfig tells us Android MDM is enabled and
	// configured.
	enterprise, err := ds.GetEnterprise(ctx)
	if err != nil {
		fmt.Println(">>>>> NO ANDROID ENTERPRISE!")
		return ctxerr.Wrap(ctx, err, "get android enterprise")
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

	// TODO(ap): at this point, we'd have a bunch of hosts that need to have their policy
	// updated. Let's simulate it for any existing Android hosts for now.
	mapIDs, err := ds.LabelIDsByName(ctx, []string{"Android"})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android label ID")
	}

	filter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
	hosts, err := ds.ListHostsInLabel(ctx, filter, mapIDs["Android"], fleet.HostListOptions{})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list android hosts")
	}

	if len(hosts) == 0 {
		fmt.Println(">>>>> NO ANDROID HOST!")
		return nil
	}

	for _, h := range hosts {
		// TODO(ap): let's use a simulated policy (that would be generated from the merged profiles)
		// for now.
		policy := &androidmanagement.Policy{
			CameraDisabled: true,
		}

		// for every policy, we want to enforce some settings
		applyFleetEnforcedSettings(policy)

		// using the host uuid as policy id, so we don't need to track the id mapping
		// to the host.
		// TODO(ap): are we seeing any downsides to this?
		policyName := fmt.Sprintf("%s/policies/%s", enterprise.Name(), h.UUID)
		applied, err := client.EnterprisesPoliciesPatch(ctx, policyName, policy)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "applying policy to host %s", h.UUID)
		}
		_ = applied
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
