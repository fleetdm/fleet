package service

import (
	"context"
	"encoding/base64"
	"strconv"
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

type pubSubPushRequest struct {
	Token                 string `query:"token"`
	android.PubSubMessage `json:"message"`
}

func pubSubPushEndpoint(ctx context.Context, request interface{}, svc android.Service) fleet.Errorer {
	req := request.(*pubSubPushRequest)
	err := svc.ProcessPubSubPush(ctx, req.Token, &req.PubSubMessage)
	return android.DefaultResponse{Err: err}
}

func (svc *Service) ProcessPubSubPush(ctx context.Context, token string, message *android.PubSubMessage) error {
	notificationType, ok := message.Attributes["notificationType"]
	level.Debug(svc.logger).Log("msg", "Received PubSub message", "notification", notificationType)
	if !ok || len(notificationType) == 0 || android.NotificationType(notificationType) == android.PubSubTest {
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
	if device.AppliedState == string(android.DeviceStateDeleted) {
		level.Debug(svc.logger).Log("msg", "Android device deleted from MDM", "device.name", device.Name,
			"device.enterpriseSpecificId", device.HardwareInfo.EnterpriseSpecificId)

		// TODO(mna): should that delete the host from Fleet? Or at least set host_mdm to unenrolled?
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

	if host != nil {
		level.Debug(svc.logger).Log("msg", "The enrolling Android host is already present in Fleet. Updating team if needed",
			"device.name", device.Name, "device.enterpriseSpecificId", device.HardwareInfo.EnterpriseSpecificId)
		enrollSecret, err := svc.ds.VerifyEnrollSecret(ctx, device.EnrollmentTokenData)
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
		return ctxerr.Errorf(ctx, "missing software info for Android device %s", device.Name)
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
		host.Device.AndroidPolicyID = policy
		host.Device.LastPolicySyncTime = ptr.Time(policySyncTime)
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

	err = svc.ds.UpdateAndroidHost(ctx, host, fromEnroll)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "enrolling Android host")
	}
	return nil
}

func (svc *Service) addNewHost(ctx context.Context, device *androidmanagement.Device) error {
	enrollSecret, err := svc.ds.VerifyEnrollSecret(ctx, device.EnrollmentTokenData)
	if err != nil && !fleet.IsNotFound(err) {
		return ctxerr.Wrap(ctx, err, "verifying enroll secret")
	}

	deviceID, err := svc.getDeviceID(ctx, device)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting device ID")
	}
	host := &fleet.AndroidHost{
		Host: &fleet.Host{
			TeamID:          enrollSecret.GetTeamID(),
			ComputerName:    svc.getComputerName(device),
			Hostname:        svc.getComputerName(device),
			Platform:        "android",
			OSVersion:       "Android " + device.SoftwareInfo.AndroidVersion,
			Build:           device.SoftwareInfo.AndroidBuildNumber,
			Memory:          device.MemoryInfo.TotalRam,
			HardwareSerial:  device.HardwareInfo.SerialNumber,
			CPUType:         device.HardwareInfo.Hardware,
			HardwareModel:   svc.getComputerName(device),
			HardwareVendor:  device.HardwareInfo.Brand,
			LabelUpdatedAt:  time.Time{},
			DetailUpdatedAt: time.Time{},
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
		host.Device.AndroidPolicyID = policy
		host.Device.LastPolicySyncTime = ptr.Time(policySyncTime)
	}
	host.SetNodeKey(device.HardwareInfo.EnterpriseSpecificId)
	_, err = svc.ds.NewAndroidHost(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "enrolling Android host")
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

func (svc *Service) getPolicyID(ctx context.Context, device *androidmanagement.Device) (*uint, error) {
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
	result, err := strconv.ParseUint(nameParts[3], 10, 64)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "parsing Android policy ID from %s", device.AppliedPolicyName)
	}
	return ptr.Uint(uint(result)), nil
}
