package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ListDevicePolicies(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
	return svc.ds.ListPoliciesForHost(ctx, host)
}

func (svc *Service) FailingPoliciesCount(ctx context.Context, host *fleet.Host) (uint, error) {
	return svc.ds.FailingPoliciesCount(ctx, host)
}

func (svc *Service) RequestEncryptionKeyRotation(ctx context.Context, hostID uint) error {
	return svc.ds.SetDiskEncryptionResetStatus(ctx, hostID, true)
}

type migrateMDMDevicePayload struct {
	Timestamp time.Time `json:"timestamp"`
	Host      struct {
		ID             uint   `json:"id"`
		UUID           string `json:"uuid"`
		HardwareSerial string `json:"hardware_serial"`
	} `json:"host"`
}

func (svc *Service) TriggerMigrateMDMDevice(ctx context.Context, host *fleet.Host) error {
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	if !ac.MDM.EnabledAndConfigured {
		return fleet.NewMDMNotConfiguredError()
	}
	if ac.MDM.MacOSMigration.WebhookURL == "" {
		return fleet.NewBadGatewayError("migrate mdm webhook", ctxerr.New(ctx, "no webhook URL configured"))
	}

	p := migrateMDMDevicePayload{}
	p.Timestamp = time.Now().UTC()
	p.Host.ID = host.ID
	p.Host.UUID = host.UUID
	p.Host.HardwareSerial = host.HardwareSerial

	if err := server.PostJSONWithTimeout(ctx, ac.MDM.MacOSMigration.WebhookURL, p); err != nil {
		return err
	}

	return nil
}
