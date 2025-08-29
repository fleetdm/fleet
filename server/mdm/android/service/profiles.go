package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

// TODO(ap): using fleet.Datastore for now as I list all hosts by label, but
// could eventually be fleet.AndroidDatastore (if that's still a thing).
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
	//
	// The profiles to apply should have status=NULL at this point, and will switch
	// to explicit status=Pending after the API requests (or Failed if there is a
	// profile overridden with another). On the pubsub status report, it will transition
	// to Verified.

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
		return nil
	}

	for _, h := range hosts {
		// TODO(ap): let's use a simulated policy (that would be generated from the merged profiles)
		// for now.
		policy := &androidmanagement.Policy{
			CameraDisabled: true,
			FunDisabled:    false,
		}

		// for every policy, we want to enforce some settings
		applyFleetEnforcedSettings(policy)

		// using the host uuid as policy id, so we don't need to track the id mapping
		// to the host.
		// TODO(ap): are we seeing any downsides to this?
		policyName := fmt.Sprintf("%s/policies/%s", enterprise.Name(), h.UUID)
		skip, err := patchPolicy(ctx, client, ds, h.UUID, policyName, policy)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "patch policy for host %d", h.ID)
		}
		if skip {
			continue
		}

		androidHost, err := ds.AndroidHostLiteByHostID(ctx, h.ID)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "get android host by host ID %d", h.ID)
		}
		if androidHost.AppliedPolicyID != nil && *androidHost.AppliedPolicyID == h.UUID {
			// that policy name is already applied to this host, it will pick up the new version
			// (confirmed in tests)
			continue
		}

		deviceName := fmt.Sprintf("%s/devices/%s", enterprise.Name(), androidHost.DeviceID)
		device := &androidmanagement.Device{
			PolicyName: policyName,
			// State must be specified when updating a device, otherwise it fails with
			// "Illegal state transition from ACTIVE to DEVICE_STATE_UNSPECIFIED"
			// TODO: should we send whatever the previous state was? If it was DISABLED,
			// we probably don't want to re-enable it by accident. Those are the only
			// 2 valid states when patching a device.
			//
			// > Note that when calling enterprises.devices.patch, ACTIVE and
			// > DISABLED are the only allowable values.
			State: "ACTIVE",
		}
		_, err = patchDevice(ctx, client, ds, h.UUID, deviceName, device)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "patch device for host %d", h.ID)
		}

		// From what I can see in tests, after the PATCH /devices, the device
		// returned will still have the old applied policy/version in the Applied
		// fields, but the PolicyName will be the new one (presumably pending
		// status report that reports the new policy as applied). Confirmed by a
		// subsequent run that returned the new policy as applied, with the
		// expected version. Note that with "funDisabled: true", I did get a
		// NonComplianceDetails for it with reason "MANAGEMENT_MODE", but the field
		// PolicyCompliant was still true.
	}

	return nil
}

func patchPolicy(ctx context.Context, client androidmgmt.Client, ds fleet.Datastore,
	policyID, policyName string, policy *androidmanagement.Policy) (skip bool, err error) {
	policyRequest, err := newAndroidPolicyRequest(policyID, policyName, policy)
	if err != nil {
		return false, ctxerr.Wrapf(ctx, err, "prepare policy request %s", policyName)
	}

	applied, apiErr := client.EnterprisesPoliciesPatch(ctx, policyName, policy)
	if apiErr != nil {
		var gerr *googleapi.Error
		if errors.As(apiErr, &gerr) {
			policyRequest.StatusCode = gerr.Code
		}
		policyRequest.ErrorDetails.V = apiErr.Error()
		policyRequest.ErrorDetails.Valid = true

		// Note that from my tests, the "not modified" error is not reliable, the
		// AMAPI happily returned 200 even if the policy was the same (as
		// confirmed by the same version number being returned), so we do check
		// for this error, but do not build critical logic on top of it.
		//
		// Tests do show that the version number is properly incremented when the
		// policy changes, though.
		if skip = androidmgmt.IsNotModifiedError(apiErr); skip {
			apiErr = nil
		}
	} else {
		policyRequest.StatusCode = 200
		policyRequest.PolicyVersion.V = applied.Version
		policyRequest.PolicyVersion.Valid = true
	}

	if err := ds.NewAndroidPolicyRequest(ctx, policyRequest); err != nil {
		return false, ctxerr.Wrap(ctx, err, "save android policy request")
	}
	return skip, ctxerr.Wrapf(ctx, apiErr, "patch policy api request failed for %s", policyName)
}

func newAndroidPolicyRequest(policyID, policyName string, policy *androidmanagement.Policy) (*fleet.MDMAndroidPolicyRequest, error) {
	b, err := json.Marshal(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy to json: %w", err)
	}
	return &fleet.MDMAndroidPolicyRequest{
		RequestName: policyName,
		PolicyID:    policyID,
		Payload:     b,
	}, nil
}

func patchDevice(ctx context.Context, client androidmgmt.Client, ds fleet.Datastore,
	policyID, deviceName string, device *androidmanagement.Device) (skip bool, apiErr error) {
	deviceRequest, err := newAndroidDeviceRequest(policyID, deviceName, device)
	if err != nil {
		return false, ctxerr.Wrapf(ctx, err, "prepare device request %s", deviceName)
	}

	applied, apiErr := client.EnterprisesDevicesPatch(ctx, deviceName, device)
	if apiErr != nil {
		var gerr *googleapi.Error
		if errors.As(apiErr, &gerr) {
			deviceRequest.StatusCode = gerr.Code
		}
		deviceRequest.ErrorDetails.V = apiErr.Error()
		deviceRequest.ErrorDetails.Valid = true

		if skip = androidmgmt.IsNotModifiedError(apiErr); skip {
			apiErr = nil
		}
	} else {
		deviceRequest.StatusCode = 200
		deviceRequest.AppliedPolicyVersion.V = applied.AppliedPolicyVersion
		deviceRequest.AppliedPolicyVersion.Valid = true
	}

	if err := ds.NewAndroidPolicyRequest(ctx, deviceRequest); err != nil {
		return false, ctxerr.Wrap(ctx, err, "save android device request")
	}
	return skip, ctxerr.Wrapf(ctx, apiErr, "patch device api request failed for %s", deviceName)
}

func newAndroidDeviceRequest(policyID, deviceName string, device *androidmanagement.Device) (*fleet.MDMAndroidPolicyRequest, error) {
	b, err := json.Marshal(device)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal device to json: %w", err)
	}
	return &fleet.MDMAndroidPolicyRequest{
		RequestName: deviceName,
		PolicyID:    policyID,
		Payload:     b,
	}, nil
}

func applyFleetEnforcedSettings(policy *androidmanagement.Policy) {
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
		ApplicationReportsEnabled:    true,
		ApplicationReportingSettings: nil, // only option is "includeRemovedApps", which I opted not to enable (we can diff apps to see removals)
	}
}
