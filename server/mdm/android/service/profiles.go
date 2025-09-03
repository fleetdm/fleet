package service

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
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
		return ctxerr.Wrap(ctx, err, "get android enterprise")
	}

	// get the list of hosts that need to have their profiles applied
	// TODO(ap): for hosts without profiles, we need the exact profiles to remove, as we
	// will have to update the host_mdm_android_profiles table to set operation remove and
	// status pending, until the pub-sub status report that will delete the row.
	hostsApplicableProfiles, hostsWithoutProfiles, err := ds.ListMDMAndroidProfilesToSend(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "identify android profiles to send")
	}

	profilesByHostUUID := make(map[string][]*fleet.MDMAndroidProfilePayload)
	profilesToLoad := make(map[string]struct{})
	for _, hostProf := range hostsApplicableProfiles {
		profilesByHostUUID[hostProf.HostUUID] = append(profilesByHostUUID[hostProf.HostUUID], hostProf)

		// keep a deduplicated list of profiles to load the JSON only once for each
		// distinct one
		profilesToLoad[hostProf.ProfileUUID] = struct{}{}
	}

	profilesContents, err := ds.GetMDMAndroidProfilesContents(ctx, slices.Collect(maps.Keys(profilesToLoad)))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load android profiles content")
	}

	// TODO(ap): a way to mock the client for tests, maybe via the license key or context? Or just an env var.
	client := newAMAPIClient(ctx, logger, licenseKey)

	// for each host, send the merged policy
	for _, hostUUID := range hostsWithoutProfiles {
		if err := sendHostProfiles(ctx, ds, client, hostUUID, nil, nil); err != nil {
		}
	}
	for hostUUID, profiles := range profilesByHostUUID {
		if err := sendHostProfiles(ctx, ds, client, hostUUID, profiles, profilesContents); err != nil {
		}
	}

	// TODO(ap): The profiles to apply should (may?) have status=NULL at this
	// point, and will switch to explicit status=Pending after the API requests
	// (or Failed if there is a profile overridden with another). On the pubsub
	// status report, it will transition to Verified.

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

	// for _, h := range hosts {
	// 	// TODO(ap): let's use a simulated policy (that would be generated from the merged profiles)
	// 	// for now.
	// 	policy := &androidmanagement.Policy{
	// 		CameraDisabled: true,
	// 		FunDisabled:    false,
	// 	}
	//
	// 	// for every policy, we want to enforce some settings
	// 	applyFleetEnforcedSettings(policy)
	//
	// 	// using the host uuid as policy id, so we don't need to track the id mapping
	// 	// to the host.
	// 	// TODO(ap): are we seeing any downsides to this?
	// 	policyName := fmt.Sprintf("%s/policies/%s", enterprise.Name(), h.UUID)
	// 	skip, err := patchPolicy(ctx, client, ds, h.UUID, policyName, policy)
	// 	if err != nil {
	// 		return ctxerr.Wrapf(ctx, err, "patch policy for host %d", h.ID)
	// 	}
	// 	if skip {
	// 		continue
	// 	}
	//
	// 	androidHost, err := ds.AndroidHostLiteByHostID(ctx, h.ID)
	// 	if err != nil {
	// 		return ctxerr.Wrapf(ctx, err, "get android host by host ID %d", h.ID)
	// 	}
	// 	if androidHost.AppliedPolicyID != nil && *androidHost.AppliedPolicyID == h.UUID {
	// 		// that policy name is already applied to this host, it will pick up the new version
	// 		// (confirmed in tests)
	// 		continue
	// 	}
	//
	// 	deviceName := fmt.Sprintf("%s/devices/%s", enterprise.Name(), androidHost.DeviceID)
	// 	device := &androidmanagement.Device{
	// 		PolicyName: policyName,
	// 		// State must be specified when updating a device, otherwise it fails with
	// 		// "Illegal state transition from ACTIVE to DEVICE_STATE_UNSPECIFIED"
	// 		// TODO: should we send whatever the previous state was? If it was DISABLED,
	// 		// we probably don't want to re-enable it by accident. Those are the only
	// 		// 2 valid states when patching a device.
	// 		//
	// 		// > Note that when calling enterprises.devices.patch, ACTIVE and
	// 		// > DISABLED are the only allowable values.
	// 		State: "ACTIVE",
	// 	}
	// 	_, err = patchDevice(ctx, client, ds, h.UUID, deviceName, device)
	// 	if err != nil {
	// 		return ctxerr.Wrapf(ctx, err, "patch device for host %d", h.ID)
	// 	}
	//
	// 	// From what I can see in tests, after the PATCH /devices, the device
	// 	// returned will still have the old applied policy/version in the Applied
	// 	// fields, but the PolicyName will be the new one (presumably pending
	// 	// status report that reports the new policy as applied). Confirmed by a
	// 	// subsequent run that returned the new policy as applied, with the
	// 	// expected version. Note that with "funDisabled: true", I did get a
	// 	// NonComplianceDetails for it with reason "MANAGEMENT_MODE", but the field
	// 	// PolicyCompliant was still true.
	// }
	_ = enterprise

	return nil
}

func sendHostProfiles(ctx context.Context, ds fleet.Datastore, client androidmgmt.Client,
	hostUUID string, profiles []*fleet.MDMAndroidProfilePayload, profilesContents map[string]json.RawMessage) error {

	// We need a deterministic order to merge the profiles, and I opted to go
	// by name, alphabetically ascending, as it's simple, deterministic (names
	// are unique) and the ordering can be viewed by the user in the UI. We had
	// also discussed upload time of the profile but it may not be
	// deterministic for batch-set profiles (same timestamp when inserted in a
	// transaction) and is not readily visible in the UI.
	slices.SortFunc(profiles, func(a, b *fleet.MDMAndroidProfilePayload) int {
		return cmp.Compare(a.ProfileName, b.ProfileName)
	})

	// merge the profiles in that order, keeping track of what profile overrides
	// what other one
	settingFromProfile := make(map[string]string) // setting name -> "winning" profile UUID
	var finalJSON map[string]json.RawMessage
	for _, prof := range profiles {
		content, ok := profilesContents[prof.ProfileUUID]
		if !ok {
			// should never happen
			return ctxerr.Errorf(ctx, "missing content for profile %s", prof.ProfileUUID)
		}

		var profJSON map[string]json.RawMessage
		if err := json.Unmarshal(content, &profJSON); err != nil {
			return ctxerr.Wrapf(ctx, err, "unmarshal profile %s content", prof.ProfileUUID)
		}

		if finalJSON == nil {
			finalJSON = profJSON
			for k := range profJSON {
				settingFromProfile[k] = prof.ProfileUUID
			}
			continue
		}

		for k, v := range profJSON {
			if _, alreadySet := finalJSON[k]; alreadySet {
				// TODO: mark settingFromProfile[k] as failed/overridden
			}
			finalJSON[k] = v
			settingFromProfile[k] = prof.ProfileUUID
		}
	}
	panic("unimplemented")
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
