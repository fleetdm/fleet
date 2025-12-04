package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

func newAndroidDeviceRequest(policyID, deviceName string, device *androidmanagement.Device) (*android.MDMAndroidPolicyRequest, error) {
	b, err := json.Marshal(device)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal device to json: %w", err)
	}
	return &android.MDMAndroidPolicyRequest{
		RequestName: deviceName,
		PolicyID:    policyID,
		Payload:     b,
	}, nil
}

func newAndroidPolicyApplicationsRequest(policyID, policyName string, apps []*androidmanagement.ApplicationPolicy) (*android.MDMAndroidPolicyRequest, error) {
	var changes []*androidmanagement.ApplicationPolicyChange
	for _, app := range apps {
		changes = append(changes, &androidmanagement.ApplicationPolicyChange{
			Application: app,
		})
	}
	req := androidmanagement.ModifyPolicyApplicationsRequest{
		Changes: changes,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal modify policy applications to json: %w", err)
	}
	return &android.MDMAndroidPolicyRequest{
		RequestName: policyName,
		PolicyID:    policyID,
		Payload:     b,
	}, nil
}

func newAndroidPolicyRequest(policyID, policyName string, policy *androidmanagement.Policy, metadata map[string]string) (*android.MDMAndroidPolicyRequest, error) {
	// save the payload with metadata about what setting comes from what profile
	m := fleet.AndroidPolicyRequestPayload{
		Policy: policy,
		Metadata: fleet.AndroidPolicyRequestPayloadMetadata{
			SettingsOrigin: metadata,
		},
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy to json: %w", err)
	}
	return &android.MDMAndroidPolicyRequest{
		RequestName: policyName,
		PolicyID:    policyID,
		Payload:     b,
	}, nil
}

// record an Android API request result in the database, filling the pre-initialized
// (via newAndroidXxxRequest) requestObject with data from the result (success or
// error). Only one of policyResult or deviceResult should be non-nil, depending
// on the type of request made.
func recordAndroidRequestResult(ctx context.Context, ds fleet.Datastore, requestObject *android.MDMAndroidPolicyRequest,
	policyResult *androidmanagement.Policy, deviceResult *androidmanagement.Device, apiErr error) (skip bool, err error) {
	if apiErr != nil {
		var gerr *googleapi.Error
		if errors.As(apiErr, &gerr) {
			requestObject.StatusCode = gerr.Code
		}
		requestObject.ErrorDetails.V = apiErr.Error()
		requestObject.ErrorDetails.Valid = true

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
		requestObject.StatusCode = http.StatusOK
		if policyResult != nil {
			requestObject.PolicyVersion.V = policyResult.Version
			requestObject.PolicyVersion.Valid = true
		} else if deviceResult != nil {
			requestObject.AppliedPolicyVersion.V = deviceResult.AppliedPolicyVersion
			requestObject.AppliedPolicyVersion.Valid = true
		}
	}

	if err := ds.NewAndroidPolicyRequest(ctx, requestObject); err != nil {
		return false, ctxerr.Wrap(ctx, err, "save android policy request")
	}
	return skip, nil
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
