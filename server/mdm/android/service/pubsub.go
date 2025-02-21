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
	return defaultResponse{Err: err}
}

func (svc *Service) ProcessPubSubPush(ctx context.Context, token string, message *android.PubSubMessage) error {
	svc.authz.SkipAuthorization(ctx)

	// TODO(26219): Verify the token

	notificationType := message.Attributes["notificationType"]
	level.Debug(svc.logger).Log("msg", "Received PubSub message", "notification", notificationType)
	if len(notificationType) == 0 {
		// Nothing to process
		return nil
	}

	var rawData []byte
	if len(message.Data) > 0 {
		var err error
		rawData, err = base64.StdEncoding.DecodeString(message.Data)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "base64 decode message.data")
		}
	}

	switch notificationType {
	case android.PubSubEnrollment:
		var device androidmanagement.Device
		err := json.Unmarshal(rawData, &device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "unmarshal Android enrollment message")
		}
		err = svc.enrollHost(ctx, &device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "enrolling Android host")
		}
	case android.PubSubStatusReport:
		var device androidmanagement.Device
		err := json.Unmarshal(rawData, &device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "unmarshal Android status report message")
		}
		host, err := svc.getExistingHost(ctx, &device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting existing Android host")
		}
		if host == nil {
			// Device is not in Fleet. Perhaps it was deleted in Fleet, but it is still connected via MDM.
			err = svc.enrollHost(ctx, &device)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "re-enrolling deleted Android host")
			}
		}
		err = svc.updateHost(ctx, &device, host)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "enrolling Android host")
		}
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

	if host != nil {
		// Update the team on the host if the host enrolled with a different team
		enrollSecret, err := svc.fleetDS.VerifyEnrollSecret(ctx, device.EnrollmentTokenData)
		if err != nil && !fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx, err, "verifying enroll secret")
		}
		host.TeamID = enrollSecret.GetTeamID()

		return svc.updateHost(ctx, device, host)
	}

	// Device is new to Fleet
	return svc.addNewHost(ctx, device)
}

func (svc *Service) getExistingHost(ctx context.Context, device *androidmanagement.Device) (*fleet.AndroidHost, error) {
	deviceID, err := svc.getDeviceID(ctx, device)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting device ID")
	}
	host, err := svc.getHostIfPresent(ctx, deviceID)
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

func (svc *Service) updateHost(ctx context.Context, device *androidmanagement.Device, host *fleet.AndroidHost) error {
	if device.AppliedPolicyName != "" {
		policy, err := svc.getPolicyID(ctx, device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting Android policy ID")
		}
		policySyncTime, err := time.Parse(time.RFC3339, device.LastPolicySyncTime)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "parsing Android policy sync time")
		}
		host.Device.PolicyID = ptr.Uint(policy)
		host.Device.LastPolicySyncTime = ptr.Time(policySyncTime)
	}

	host.Host.ComputerName = svc.getComputerName(device)
	host.Host.Hostname = svc.getComputerName(device)
	host.Host.OSVersion = "Android " + device.SoftwareInfo.AndroidVersion
	host.Host.Build = device.SoftwareInfo.AndroidBuildNumber
	host.Host.Memory = device.MemoryInfo.TotalRam
	host.Host.HardwareSerial = device.HardwareInfo.SerialNumber
	host.Device.EnterpriseSpecificID = ptr.String(device.HardwareInfo.EnterpriseSpecificId)
	host.LabelUpdatedAt = time.Time{}
	if device.LastStatusReportTime != "" {
		lastStatusReportTime, err := time.Parse(time.RFC3339, device.LastStatusReportTime)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "parsing Android last status report time")
		}
		host.DetailUpdatedAt = lastStatusReportTime
	}

	err := svc.fleetDS.UpdateAndroidHost(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "enrolling Android host")
	}
	return nil
}

func (svc *Service) addNewHost(ctx context.Context, device *androidmanagement.Device) error {
	enrollSecret, err := svc.fleetDS.VerifyEnrollSecret(ctx, device.EnrollmentTokenData)
	if err != nil && !fleet.IsNotFound(err) {
		return ctxerr.Wrap(ctx, err, "verifying enroll secret")
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
			LabelUpdatedAt:  time.Time{},
			DetailUpdatedAt: time.Time{},
		},
		Device: &android.Device{
			EnterpriseSpecificID: ptr.String(device.HardwareInfo.EnterpriseSpecificId),
		},
	}
	deviceID, err := svc.getDeviceID(ctx, device)
	if device.AppliedPolicyName != "" {
		policy, err := svc.getPolicyID(ctx, device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting Android policy ID")
		}
		policySyncTime, err := time.Parse(time.RFC3339, device.LastPolicySyncTime)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "parsing Android policy sync time")
		}
		host.Device.PolicyID = ptr.Uint(policy)
		host.Device.LastPolicySyncTime = ptr.Time(policySyncTime)
	}

	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting device ID")
	}
	host.SetDeviceID(deviceID)
	_, err = svc.fleetDS.NewAndroidHost(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "enrolling Android host")
	}
	return nil
}

func (svc *Service) getComputerName(device *androidmanagement.Device) string {
	computerName := cases.Title(language.English, cases.Compact).String(device.HardwareInfo.Brand) + " " + device.HardwareInfo.Model
	return computerName
}

func (svc *Service) getHostIfPresent(ctx context.Context, deviceID string) (*fleet.AndroidHost, error) {
	host, err := svc.fleetDS.AndroidHostLite(ctx, deviceID)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "getting device by device ID")
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

func (svc *Service) getPolicyID(ctx context.Context, device *androidmanagement.Device) (uint, error) {
	nameParts := strings.Split(device.AppliedPolicyName, "/")
	if len(nameParts) != 4 {
		return 0, ctxerr.Errorf(ctx, "invalid Android policy name: %s", device.AppliedPolicyName)
	}
	result, err := strconv.ParseUint(nameParts[3], 10, 64)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "parsing Android policy ID")
	}
	return uint(result), nil
}
