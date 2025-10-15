package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-json-experiment/json"
	"github.com/go-kit/log/level"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/api/androidmanagement/v1"
)

type PubSubPushRequest struct {
	Token                 string `query:"token"`
	android.PubSubMessage `json:"message"`
}

func pubSubPushEndpoint(ctx context.Context, request interface{}, svc android.Service) fleet.Errorer {
	req := request.(*PubSubPushRequest)
	err := svc.ProcessPubSubPush(ctx, req.Token, &req.PubSubMessage)
	return android.DefaultResponse{Err: err}
}

func (svc *Service) ProcessPubSubPush(ctx context.Context, token string, message *android.PubSubMessage) error {
	notificationType, ok := message.Attributes["notificationType"]
	if !ok || len(notificationType) == 0 {
		// Nothing to process
		svc.authz.SkipAuthorization(ctx)
		return nil
	}
	level.Debug(svc.logger).Log("msg", "Received PubSub message", "notification", notificationType)
	if android.NotificationType(notificationType) == android.PubSubTest {
		// Nothing to process
		svc.authz.SkipAuthorization(ctx)
		return nil
	}

	var rawData []byte
	if len(message.Data) > 0 {
		var err error
		rawData, err = base64.StdEncoding.DecodeString(message.Data)
		if err != nil {
			svc.authz.SkipAuthorization(ctx)
			return ctxerr.Wrap(ctx, err, "base64 decode message.data")
		}
	}

	switch android.NotificationType(notificationType) {
	case android.PubSubEnrollment:
		return svc.handlePubSubEnrollment(ctx, token, rawData)
	case android.PubSubStatusReport:
		return svc.handlePubSubStatusReport(ctx, token, rawData)
	default:
		// Ignore unknown notification types
		level.Debug(svc.logger).Log("msg", "Ignoring PubSub notification type", "notification", notificationType)
		svc.authz.SkipAuthorization(ctx)
		return nil
	}
}

func (svc *Service) authenticatePubSub(ctx context.Context, token string) error {
	svc.authz.SkipAuthorization(ctx)
	_, err := svc.checkIfAndroidNotConfigured(ctx)
	if err != nil {
		return err
	}

	// Verify the token
	//
	// GetAllMDMConfigAssetsByName does one DB read of the hash, but decrypted asset value is cached, so we don't pay the CPU decryption cost.
	// If this `mdm_config_assets` access becomes a bottleneck, we can cache the decrypted value without re-checking the hash.
	//
	// Note: We could also check that the device belongs to our enterprise, for additional security. We would need an Android cached_mysql for that.
	// "name": "enterprises/LC044q09r2/devices/3dc9d72fbd517bbc",
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidPubSubToken}, nil)
	switch {
	case fleet.IsNotFound(err):
		return fleet.NewAuthFailedError("missing Android PubSub token in Fleet")
	case err != nil:
		return ctxerr.Wrap(ctx, err, "getting Android PubSub token")
	}
	goldenToken, ok := assets[fleet.MDMAssetAndroidPubSubToken]
	if !ok || string(goldenToken.Value) != token {
		return fleet.NewAuthFailedError("invalid Android PubSub token")
	}
	return nil
}

func (svc *Service) getClientAuthenticationSecret(ctx context.Context) (string, error) {
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidFleetServerSecret}, nil)
	switch {
	case fleet.IsNotFound(err):
		return "", nil
	case err != nil:
		return "", ctxerr.Wrap(ctx, err, "getting Android authentication secret")
	}
	return string(assets[fleet.MDMAssetAndroidFleetServerSecret].Value), nil
}

func (svc *Service) handlePubSubStatusReport(ctx context.Context, token string, rawData []byte) error {
	// We allow DELETED notification type to be received since user may be in the process of disabling Android MDM.
	// Otherwise, we authenticate below in authenticatePubSub
	svc.authz.SkipAuthorization(ctx)

	var device androidmanagement.Device
	err := json.Unmarshal(rawData, &device)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal Android status report message")
	}

	// Consider both appliedState and state fields for deletion, to handle variations in payloads.
	isDeleted := strings.ToUpper(device.AppliedState) == string(android.DeviceStateDeleted)
	if !isDeleted {
		var alt struct {
			AppliedState string `json:"appliedState"`
			State        string `json:"state"`
		}
		// Best-effort parse; ignore error if shape doesn't match.
		_ = json.Unmarshal(rawData, &alt)
		if strings.ToUpper(alt.AppliedState) == string(android.DeviceStateDeleted) || strings.ToUpper(alt.State) == string(android.DeviceStateDeleted) {
			isDeleted = true
		}
	}

	if isDeleted {
		level.Debug(svc.logger).Log("msg", "Android device deleted from MDM", "device.name", device.Name,
			"device.enterpriseSpecificId", device.HardwareInfo.EnterpriseSpecificId)

		// User-initiated unenroll (work profile removed) or device deleted via AMAPI.
		// Flip host_mdm to unenrolled and emit an activity.
		host, err := svc.getExistingHost(ctx, &device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get host for deleted android device")
		}
		if host != nil {
			didUnenroll, err := svc.ds.SetAndroidHostUnenrolled(ctx, host.Host.ID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "set android host unenrolled on DELETED state")
			}
			if !didUnenroll {
				return nil // Skip activity, if we didn't update the enrollment state.
			}

			// Emit system activity: mdm_unenrolled. For Android BYOD, InstalledFromDEP is always false.
			// Use the computed display name from the device payload as lite host may not include it.
			displayName := svc.getComputerName(&device)
			_ = svc.fleetSvc.NewActivity(ctx, nil, fleet.ActivityTypeMDMUnenrolled{
				HostSerial:       "",
				HostDisplayName:  displayName,
				InstalledFromDEP: false,
				Platform:         host.Platform,
			})
		}
		return nil
	}

	err = svc.authenticatePubSub(ctx, token)
	if err != nil {
		return err
	}

	host, err := svc.getExistingHost(ctx, &device)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting existing Android host")
	}
	if host == nil {
		level.Debug(svc.logger).Log("msg", "Device not found in Fleet. Perhaps it was deleted, "+
			"but it is still connected via Android MDM. Re-enrolling", "device.name", device.Name,
			"device.enterpriseSpecificId", device.HardwareInfo.EnterpriseSpecificId)
		err = svc.enrollHost(ctx, &device)
		if err != nil {
			level.Debug(svc.logger).Log("msg", "Error re-enrolling Android host", "data", rawData)
			return ctxerr.Wrap(ctx, err, "re-enrolling deleted Android host")
		}
	}
	err = svc.updateHost(ctx, &device, host, false)
	if err != nil {
		level.Debug(svc.logger).Log("msg", "Error updating Android host", "data", rawData)
		return ctxerr.Wrap(ctx, err, "enrolling Android host")
	}
	err = svc.updateHostSoftware(ctx, &device, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating Android host software")
	}
	return nil
}

// largely based on refetch apps code from Apple MDM service methods
func (svc *Service) updateHostSoftware(ctx context.Context, device *androidmanagement.Device, host *fleet.AndroidHost) error {
	// Do nothing if no app reports returned
	if len(device.ApplicationReports) == 0 {
		return nil
	}
	truncateString := func(item any, length int) string {
		str, ok := item.(string)
		if !ok {
			return ""
		}
		runes := []rune(str)
		if len(runes) > length {
			return string(runes[:length])
		}
		return str
	}
	software := []fleet.Software{}
	for _, app := range device.ApplicationReports {
		if app.State != "INSTALLED" {
			continue
		}
		sw := fleet.Software{
			Name:          truncateString(app.DisplayName, fleet.SoftwareNameMaxLength),
			Version:       truncateString(app.VersionName, fleet.SoftwareVersionMaxLength),
			ApplicationID: ptr.String(truncateString(app.PackageName, fleet.SoftwareBundleIdentifierMaxLength)),
			Source:        "android_apps",
			Installed:     true,
		}
		software = append(software, sw)
	}

	_, err := svc.fleetDS.UpdateHostSoftware(ctx, host.Host.ID, software)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating Android host software")
	}
	return nil
}

func (svc *Service) handlePubSubEnrollment(ctx context.Context, token string, rawData []byte) error {
	err := svc.authenticatePubSub(ctx, token)
	if err != nil {
		return err
	}

	var device androidmanagement.Device
	err = json.Unmarshal(rawData, &device)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal Android enrollment message")
	}

	// Some deployments may report work profile removal under ENROLLMENT notifications.
	// Detect DELETED here too and treat as unenrollment confirmation.
	isDeleted := strings.ToUpper(device.AppliedState) == string(android.DeviceStateDeleted)
	if !isDeleted {
		var alt struct {
			AppliedState string `json:"appliedState"`
			State        string `json:"state"`
		}
		_ = json.Unmarshal(rawData, &alt)
		if strings.ToUpper(alt.AppliedState) == string(android.DeviceStateDeleted) || strings.ToUpper(alt.State) == string(android.DeviceStateDeleted) {
			isDeleted = true
		}
	}
	if isDeleted {
		// Bypass re-enrollment and flip host to unenrolled.
		host, herr := svc.getExistingHost(ctx, &device)
		if herr != nil {
			return ctxerr.Wrap(ctx, herr, "get host for deleted android device (ENROLLMENT)")
		}
		if host != nil {
			if _, err := svc.ds.SetAndroidHostUnenrolled(ctx, host.Host.ID); err != nil {
				return ctxerr.Wrap(ctx, err, "set android host unenrolled on DELETED state (ENROLLMENT)")
			}
			displayName := svc.getComputerName(&device)
			_ = svc.fleetSvc.NewActivity(ctx, nil, fleet.ActivityTypeMDMUnenrolled{
				HostSerial:       "",
				HostDisplayName:  displayName,
				InstalledFromDEP: false,
				Platform:         host.Platform,
			})
		}
		return nil
	}

	err = svc.enrollHost(ctx, &device)
	if err != nil {
		level.Debug(svc.logger).Log("msg", "Error enrolling Android host", "data", rawData)
		return ctxerr.Wrap(ctx, err, "enrolling Android host")
	}
	return nil
}

func (svc *Service) enrollHost(ctx context.Context, device *androidmanagement.Device) error {
	err := svc.validateDevice(ctx, device)
	if err != nil {
		return err
	}

	// Device may already be present in Fleet if device user removed the MDM profile and then re-enrolled
	host, err := svc.getExistingHost(ctx, device)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting existing Android host")
	}

	// TODO(mna): in the next iteration of Android work (as we're short on time
	// to make it in the release), we should refactor this to use the MDM
	// lifecycle and update the lifecycle to support Android, so that TurnOnMDM
	// inserts the host_mdm, and TurnOffMDM deletes it.

	var enrollmentTokenRequest enrollmentTokenRequest
	err = json.Unmarshal([]byte(device.EnrollmentTokenData), &enrollmentTokenRequest)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshilling enrollment token data")
	}

	if host != nil {
		level.Debug(svc.logger).Log("msg", "The enrolling Android host is already present in Fleet. Updating team if needed",
			"device.name", device.Name, "device.enterpriseSpecificId", device.HardwareInfo.EnterpriseSpecificId)
		enrollSecret, err := svc.ds.VerifyEnrollSecret(ctx, enrollmentTokenRequest.EnrollSecret)
		if err != nil && !fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx, err, "verifying enroll secret")
		}
		host.TeamID = enrollSecret.GetTeamID()

		return svc.updateHost(ctx, device, host, true)
	}

	// Device is new to Fleet
	return svc.addNewHost(ctx, device)
}

func (svc *Service) getExistingHost(ctx context.Context, device *androidmanagement.Device) (*fleet.AndroidHost, error) {
	host, err := svc.getHostIfPresent(ctx, device.HardwareInfo.EnterpriseSpecificId)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting Android host if present")
	}
	return host, nil
}

func (svc *Service) validateDevice(ctx context.Context, device *androidmanagement.Device) error {
	if device.HardwareInfo == nil {
		return ctxerr.Errorf(ctx, "missing hardware info for Android device %s", device.Name)
	}
	if device.SoftwareInfo == nil {
		return ctxerr.Errorf(ctx, "missing software info for Android device %s. Are policy statusReportingSettings set correctly?", device.Name)
	}
	if device.MemoryInfo == nil {
		return ctxerr.Errorf(ctx, "missing memory info for Android device %s", device.Name)
	}
	return nil
}

func (svc *Service) updateHost(ctx context.Context, device *androidmanagement.Device, host *fleet.AndroidHost, fromEnroll bool) error {
	err := svc.validateDevice(ctx, device)
	if err != nil {
		return err
	}
	if device.AppliedPolicyName != "" {
		policy, err := svc.getPolicyID(ctx, device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting Android policy ID")
		}
		policySyncTime, err := time.Parse(time.RFC3339, device.LastPolicySyncTime)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "parsing Android policy sync time")
		}
		host.Device.AppliedPolicyID = policy
		if device.AppliedPolicyVersion != 0 {
			host.Device.AppliedPolicyVersion = &device.AppliedPolicyVersion
		}
		host.Device.LastPolicySyncTime = ptr.Time(policySyncTime)
		svc.verifyDevicePolicy(ctx, host.UUID, device)
	}

	deviceID, err := svc.getDeviceID(ctx, device)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting device ID")
	}
	host.Device.DeviceID = deviceID

	host.Host.ComputerName = svc.getComputerName(device)
	host.Host.Hostname = svc.getComputerName(device)
	host.Host.Platform = "android"
	host.Host.OSVersion = "Android " + device.SoftwareInfo.AndroidVersion
	host.Host.Build = device.SoftwareInfo.AndroidBuildNumber
	host.Host.Memory = device.MemoryInfo.TotalRam

	host.Host.GigsTotalDiskSpace, host.Host.GigsDiskSpaceAvailable, host.Host.PercentDiskSpaceAvailable = svc.calculateAndroidStorageMetrics(ctx, device, true)

	host.Host.HardwareSerial = device.HardwareInfo.SerialNumber
	host.Host.CPUType = device.HardwareInfo.Hardware
	host.Host.HardwareModel = svc.getComputerName(device)
	host.Host.HardwareVendor = device.HardwareInfo.Brand
	host.LabelUpdatedAt = time.Time{}
	if device.LastStatusReportTime != "" {
		lastStatusReportTime, err := time.Parse(time.RFC3339, device.LastStatusReportTime)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "parsing Android last status report time")
		}
		host.DetailUpdatedAt = lastStatusReportTime
	}
	host.SetNodeKey(device.HardwareInfo.EnterpriseSpecificId)
	if device.HardwareInfo.EnterpriseSpecificId != "" {
		host.Host.UUID = device.HardwareInfo.EnterpriseSpecificId
	}

	err = svc.ds.UpdateAndroidHost(ctx, host, fromEnroll)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "enrolling Android host")
	}
	// Enrollment activities are intentionally not emitted for Android at this time.
	return nil
}

func (svc *Service) addNewHost(ctx context.Context, device *androidmanagement.Device) error {
	var enrollmentTokenRequest enrollmentTokenRequest
	err := json.Unmarshal([]byte(device.EnrollmentTokenData), &enrollmentTokenRequest)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshilling enrollment token data")
	}

	enrollSecret, err := svc.ds.VerifyEnrollSecret(ctx, enrollmentTokenRequest.EnrollSecret)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "verifying enroll secret")
	}

	deviceID, err := svc.getDeviceID(ctx, device)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting device ID")
	}

	gigsTotalDiskSpace, gigsDiskSpaceAvailable, percentDiskSpaceAvailable := svc.calculateAndroidStorageMetrics(ctx, device, false)

	host := &fleet.AndroidHost{
		Host: &fleet.Host{
			TeamID:                    enrollSecret.GetTeamID(),
			ComputerName:              svc.getComputerName(device),
			Hostname:                  svc.getComputerName(device),
			Platform:                  "android",
			OSVersion:                 "Android " + device.SoftwareInfo.AndroidVersion,
			Build:                     device.SoftwareInfo.AndroidBuildNumber,
			Memory:                    device.MemoryInfo.TotalRam,
			GigsTotalDiskSpace:        gigsTotalDiskSpace,
			GigsDiskSpaceAvailable:    gigsDiskSpaceAvailable,
			PercentDiskSpaceAvailable: percentDiskSpaceAvailable,
			HardwareSerial:            device.HardwareInfo.SerialNumber,
			CPUType:                   device.HardwareInfo.Hardware,
			HardwareModel:             svc.getComputerName(device),
			HardwareVendor:            device.HardwareInfo.Brand,
			LabelUpdatedAt:            time.Time{},
			DetailUpdatedAt:           time.Time{},
			UUID:                      device.HardwareInfo.EnterpriseSpecificId,
		},
		Device: &android.Device{
			DeviceID: deviceID,
		},
	}
	if device.AppliedPolicyName != "" {
		policy, err := svc.getPolicyID(ctx, device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting Android policy ID")
		}
		policySyncTime, err := time.Parse(time.RFC3339, device.LastPolicySyncTime)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "parsing Android policy sync time")
		}
		host.Device.AppliedPolicyID = policy
		if device.AppliedPolicyVersion != 0 {
			host.Device.AppliedPolicyVersion = &device.AppliedPolicyVersion
		}
		host.Device.LastPolicySyncTime = ptr.Time(policySyncTime)
	}
	host.SetNodeKey(device.HardwareInfo.EnterpriseSpecificId)
	_, err = svc.ds.NewAndroidHost(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "enrolling Android host")
	}

	if enrollmentTokenRequest.IdpUUID != "" {
		level.Info(svc.logger).Log("msg", "associating android host with idp account", "host_uuid", host.UUID, "idp_uuid", enrollmentTokenRequest.IdpUUID)
		err := svc.ds.AssociateHostMDMIdPAccount(ctx, host.UUID, enrollmentTokenRequest.IdpUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "associating host with idp account")
		}
	}

	return nil
}

func (svc *Service) getComputerName(device *androidmanagement.Device) string {
	computerName := cases.Title(language.English, cases.Compact).String(device.HardwareInfo.Brand) + " " + device.HardwareInfo.Model
	return computerName
}

func (svc *Service) getHostIfPresent(ctx context.Context, enterpriseSpecificID string) (*fleet.AndroidHost, error) {
	host, err := svc.ds.AndroidHostLite(ctx, enterpriseSpecificID)
	switch {
	case fleet.IsNotFound(err):
		return nil, nil
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting Android host")
	}
	return host, nil
}

func (svc *Service) getDeviceID(ctx context.Context, device *androidmanagement.Device) (string, error) {
	nameParts := strings.Split(device.Name, "/")
	if len(nameParts) != 4 {
		return "", ctxerr.Errorf(ctx, "invalid Android device name: %s", device.Name)
	}
	deviceID := nameParts[3]
	return deviceID, nil
}

func (svc *Service) getPolicyID(ctx context.Context, device *androidmanagement.Device) (*string, error) {
	nameParts := strings.Split(device.AppliedPolicyName, "/")
	if len(nameParts) != 4 {
		return nil, ctxerr.Errorf(ctx, "invalid Android policy name: %s", device.AppliedPolicyName)
	}
	if len(nameParts[3]) == 0 {
		level.Error(svc.logger).Log("msg", "Empty Android policy ID", "device.name", device.Name,
			"device.enterpriseSpecificID", device.HardwareInfo.EnterpriseSpecificId, "device.AppliedPolicyName",
			device.AppliedPolicyName)
		return nil, nil
	}
	return ptr.String(nameParts[3]), nil
}

func (svc *Service) verifyDevicePolicy(ctx context.Context, hostUUID string, device *androidmanagement.Device) {
	appliedPolicyVersion := device.AppliedPolicyVersion

	level.Debug(svc.logger).Log("msg", "Verifying Android device policy", "host_uuid", hostUUID, "applied_policy_version", appliedPolicyVersion)

	// Get all host_mdm_android_profiles that is pending, and included_in_policy_version = device.AppliedPolicyVersion.
	// That way we can either fully verify the profile, or mark as failed if the field it tries to set is not compliant.

	// Get all profiles that are pending install
	pendingInstallProfiles, err := svc.ds.ListHostMDMAndroidProfilesPendingInstallWithVersion(ctx, hostUUID, appliedPolicyVersion)
	if err != nil {
		level.Error(svc.logger).Log("msg", "error getting pending profiles", "err", err)
		return
	}
	pendingProfilesUUIDMap := make(map[string]*fleet.MDMAndroidProfilePayload, len(pendingInstallProfiles))
	for _, profile := range pendingInstallProfiles {
		pendingProfilesUUIDMap[profile.ProfileUUID] = profile
	}

	// First case, if nonComplianceDetails is empty, verify all profiles that is pending install, and remove the pending remove ones.
	if len(device.NonComplianceDetails) == 0 {
		var verifiedProfiles []*fleet.MDMAndroidProfilePayload
		for _, profile := range pendingInstallProfiles {
			verifiedProfiles = append(verifiedProfiles, &fleet.MDMAndroidProfilePayload{
				HostUUID:                profile.HostUUID,
				Status:                  &fleet.MDMDeliveryVerified,
				OperationType:           profile.OperationType,
				ProfileUUID:             profile.ProfileUUID,
				Detail:                  profile.Detail,
				ProfileName:             profile.ProfileName,
				PolicyRequestUUID:       profile.PolicyRequestUUID,
				DeviceRequestUUID:       profile.DeviceRequestUUID,
				RequestFailCount:        profile.RequestFailCount,
				IncludedInPolicyVersion: profile.IncludedInPolicyVersion,
			})
		}

		err = svc.ds.BulkUpsertMDMAndroidHostProfiles(ctx, verifiedProfiles)
		if err != nil {
			level.Error(svc.logger).Log("msg", "error verifying pending install profiles", "err", err)
		}

	} else {
		// Dedupe the policyRequestUUID across all pending install profiles
		var policyRequestUUID string
		for _, profile := range pendingInstallProfiles {
			if int64(*profile.IncludedInPolicyVersion) == device.AppliedPolicyVersion && profile.PolicyRequestUUID != nil {
				policyRequestUUID = *profile.PolicyRequestUUID
			}
		}

		// Iterate over all policy request uuids, fetch them and unmarshal the payload into the type.
		// Then re-use the map above, so we can iterate over it again, but now the payload is already unmarshalled.
		policyRequest, err := svc.ds.GetAndroidPolicyRequestByUUID(ctx, policyRequestUUID)
		if err != nil && !fleet.IsNotFound(err) {
			level.Error(svc.logger).Log("msg", "error getting policy request", "err", err, "policy_request_uuid", policyRequestUUID, "host_uuid", hostUUID)
			return
		}

		if fleet.IsNotFound(err) {
			level.Error(svc.logger).Log("msg", "policy request not found", "policy_request_uuid", policyRequestUUID, "host_uuid", hostUUID)
			return
		}

		var policyRequestPayload fleet.AndroidPolicyRequestPayload
		err = json.Unmarshal(policyRequest.Payload, &policyRequestPayload)
		if err != nil {
			level.Error(svc.logger).Log("msg", "error unmarshalling policy request payload", "err", err, "policy_request_uuid", policyRequestUUID, "host_uuid", hostUUID)
			return
		}

		// Go over nonComplianceDetails, lookup the setting name, and get the corresponding profile based on the policyRequestPayload metadata settings origin.
		// Update the status of the profiles to failed, and add the correct detail error message.
		failedProfileUUIDsWithNonCompliances := make(map[string][]*androidmanagement.NonComplianceDetail)
		for _, nonCompliance := range device.NonComplianceDetails {
			profileUUIDToMarkAsFailed := policyRequestPayload.Metadata.SettingsOrigin[nonCompliance.SettingName]
			if _, ok := failedProfileUUIDsWithNonCompliances[profileUUIDToMarkAsFailed]; !ok {
				failedProfileUUIDsWithNonCompliances[profileUUIDToMarkAsFailed] = []*androidmanagement.NonComplianceDetail{}
			}

			failedProfileUUIDsWithNonCompliances[profileUUIDToMarkAsFailed] = append(failedProfileUUIDsWithNonCompliances[profileUUIDToMarkAsFailed], nonCompliance)
		}

		var profiles []*fleet.MDMAndroidProfilePayload
		for _, profile := range pendingInstallProfiles {
			status := &fleet.MDMDeliveryVerified
			detail := profile.Detail

			if nonCompliance, ok := failedProfileUUIDsWithNonCompliances[profile.ProfileUUID]; ok {
				status = &fleet.MDMDeliveryFailed
				detail = buildNonComplianceErrorMessage(nonCompliance)
			}

			profiles = append(profiles, &fleet.MDMAndroidProfilePayload{
				HostUUID:                profile.HostUUID,
				Status:                  status,
				ProfileUUID:             profile.ProfileUUID,
				OperationType:           profile.OperationType,
				DeviceRequestUUID:       profile.DeviceRequestUUID,
				RequestFailCount:        profile.RequestFailCount,
				IncludedInPolicyVersion: profile.IncludedInPolicyVersion,
				ProfileName:             profile.ProfileName,
				PolicyRequestUUID:       profile.PolicyRequestUUID,
				Detail:                  detail,
			})
		}

		err = svc.ds.BulkUpsertMDMAndroidHostProfiles(ctx, profiles)
		if err != nil {
			level.Error(svc.logger).Log("msg", "error upserting android profiles", "err", err, "host_uuid", hostUUID)
			return
		}
	}

	// Bulk delete any pending or failed remove profiles.
	err = svc.ds.BulkDeleteMDMAndroidHostProfiles(ctx, hostUUID, appliedPolicyVersion)
	if err != nil {
		level.Error(svc.logger).Log("msg", "error deleting pending or failed remove profiles", "err", err, "host_uuid", hostUUID)
	}
}

func buildNonComplianceErrorMessage(nonCompliance []*androidmanagement.NonComplianceDetail) string {
	failedSettings := []string{}
	failedReasons := []string{}

	for _, detail := range nonCompliance {
		failedSettings = append(failedSettings, fmt.Sprintf("%q", detail.SettingName))
		failedReasons = append(failedReasons, detail.NonComplianceReason)
	}
	failedSettingsString := strings.Join(failedSettings[:len(failedSettings)-1], ", ") + ", and " + failedSettings[len(failedSettings)-1]
	failedReasonsString := strings.Join(failedReasons[:len(failedReasons)-1], ", ") + ", and " + failedReasons[len(failedReasons)-1]

	return fmt.Sprintf("%s settings couldn't apply to a host.\nReasons: %s. Other settings are applied.", failedSettingsString, failedReasonsString)
}

// calculateAndroidStorageMetrics processes Android device memory events and calculates storage metrics.
// Returns -1 for both available space and percentage values when we don't receive the AMAPI fields needed to calculate storage.
func (svc *Service) calculateAndroidStorageMetrics(
	ctx context.Context,
	device *androidmanagement.Device,
	isUpdate bool,
) (gigsTotalDiskSpace, gigsDiskSpaceAvailable, percentDiskSpaceAvailable float64) {
	if device.MemoryInfo == nil || device.MemoryInfo.TotalInternalStorage <= 0 {
		return 0, 0, 0
	}

	totalStorageBytes := device.MemoryInfo.TotalInternalStorage

	// Determine log message prefix based on context
	logPrefix := "Processing Android memory events"
	logSuffix := ""
	if isUpdate {
		logSuffix = " (update)"
	}

	// Log memory events for debugging
	level.Debug(svc.logger).Log(
		"msg", logPrefix+logSuffix,
		"device_id", device.HardwareInfo.EnterpriseSpecificId,
		"total_internal_storage", totalStorageBytes,
		"memory_events_count", len(device.MemoryEvents),
	)

	var totalAvailableBytes int64
	var hasMeasuredEvents bool

	// Track the latest external storage detection event to avoid accumulation
	var latestExternalStorageBytes int64
	var latestExternalStorageTime time.Time

	// Track the latest measured events to avoid accumulation
	var latestInternalMeasuredBytes int64
	var latestInternalMeasuredTime time.Time
	var latestExternalMeasuredBytes int64
	var latestExternalMeasuredTime time.Time

	for _, event := range device.MemoryEvents {
		level.Debug(svc.logger).Log(
			"msg", "Android memory event"+logSuffix,
			"event_type", event.EventType,
			"byte_count", event.ByteCount,
			"create_time", event.CreateTime,
		)

		eventTime, err := time.Parse(time.RFC3339, event.CreateTime)
		if err != nil {
			// Log parse error but continue processing
			level.Debug(svc.logger).Log(
				"msg", "Failed to parse event time"+logSuffix,
				"event_type", event.EventType,
				"create_time", event.CreateTime,
				"error", err,
			)
			continue
		}

		switch event.EventType {
		case "EXTERNAL_STORAGE_DETECTED":
			// Only use the most recent EXTERNAL_STORAGE_DETECTED event
			if eventTime.After(latestExternalStorageTime) {
				latestExternalStorageBytes = event.ByteCount
				latestExternalStorageTime = eventTime
			}
		case "INTERNAL_STORAGE_MEASURED":
			// Only use the most recent INTERNAL_STORAGE_MEASURED event
			if eventTime.After(latestInternalMeasuredTime) {
				latestInternalMeasuredBytes = event.ByteCount
				latestInternalMeasuredTime = eventTime
				hasMeasuredEvents = true
			}
		case "EXTERNAL_STORAGE_MEASURED":
			// Only use the most recent EXTERNAL_STORAGE_MEASURED event
			if eventTime.After(latestExternalMeasuredTime) {
				latestExternalMeasuredBytes = event.ByteCount
				latestExternalMeasuredTime = eventTime
				hasMeasuredEvents = true
			}
		}
	}

	// Add the latest external storage value (if any) to the total
	if latestExternalStorageBytes > 0 {
		totalStorageBytes += latestExternalStorageBytes
	}

	// Calculate total available from the latest measured events
	totalAvailableBytes = latestInternalMeasuredBytes + latestExternalMeasuredBytes

	if totalStorageBytes > 0 {
		gigsTotalDiskSpace = float64(totalStorageBytes) / (1024 * 1024 * 1024)

		// If we only have DETECTED events (no MEASURED events), available space measurement isn't supported
		// We can still report total storage capacity but not how much is free/used
		// We use -1 as sentinel value to indicate "not supported"
		if !hasMeasuredEvents {
			gigsDiskSpaceAvailable = -1
			percentDiskSpaceAvailable = -1

			level.Debug(svc.logger).Log(
				"msg", "Android storage measurement not supported"+logSuffix,
				"device_id", device.HardwareInfo.EnterpriseSpecificId,
				"total_storage_bytes", totalStorageBytes,
				"reason", "Only DETECTED events, no MEASURED events",
			)
		} else {
			gigsDiskSpaceAvailable = float64(totalAvailableBytes) / (1024 * 1024 * 1024)
			percentDiskSpaceAvailable = (float64(totalAvailableBytes) / float64(totalStorageBytes)) * 100

			level.Debug(svc.logger).Log(
				"msg", "Android storage calculation complete"+logSuffix,
				"total_storage_bytes", totalStorageBytes,
				"total_available_bytes", totalAvailableBytes,
				"gigs_total", gigsTotalDiskSpace,
				"gigs_available", gigsDiskSpaceAvailable,
				"percent_available", percentDiskSpaceAvailable,
			)
		}
	}

	return gigsTotalDiskSpace, gigsDiskSpaceAvailable, percentDiskSpaceAvailable
}
