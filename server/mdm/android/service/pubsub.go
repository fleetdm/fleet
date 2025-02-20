package service

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/go-json-experiment/json"
	"github.com/go-kit/log/level"
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

	// TODO: Verify the token

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
			return ctxerr.Wrap(ctx, err, "unmarshal enrollment message")
		}
		err = svc.enroll(ctx, &device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "enrolling device")
		}
	case android.PubSubStatusReport:
		// TODO: Update device details and timestamps
	}

	return nil
}

func (svc *Service) enroll(ctx context.Context, device *androidmanagement.Device) error {

	// Device may already be present in Fleet if device user removed the MDM profile and then re-enrolled
	fleetDevice, err := svc.getDeviceIfPresent(ctx, device)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting Android device if present")
	}
	// TODO: Check if entry in hosts table is also present. If not, create one.

	if fleetDevice != nil {
		// TODO: Update device details; this could also be a re-enrollment
		// _, err := svc.fleetDS.HostLite(ctx, fleetDevice.HostID)
		// if err != nil && !fleet.IsNotFound(err) {
		// 	return ctxerr.Wrap(ctx, err, "getting host")
		// }
	} else {
		// Device is new to Fleet
		enrollSecret, err := svc.fleetDS.VerifyEnrollSecret(ctx, device.EnrollmentTokenData)
		if err != nil && !fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx, err, "verifying enroll secret")
		}
		// TODO: Do EnrollHost and androidDS.AddHost inside a transaction so we don't add duplicate hosts
		_, err = svc.fleetDS.EnrollHost(ctx, true, device.HardwareInfo.SerialNumber, device.HardwareInfo.SerialNumber,
			device.HardwareInfo.SerialNumber, "", enrollSecret.GetTeamID(), 0)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "enrolling host")
		}
	}

	return nil
}

func (svc *Service) getDeviceIfPresent(ctx context.Context, device *androidmanagement.Device) (*android.Device, error) {
	nameParts := strings.Split(device.Name, "/")
	if len(nameParts) != 4 {
		return nil, ctxerr.Errorf(ctx, "invalid Android device name: %s", device.Name)
	}
	deviceID := nameParts[3]
	fleetDevice, err := svc.ds.GetDeviceByDeviceID(ctx, deviceID)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "getting device by device ID")
	}
	return fleetDevice, nil
}
