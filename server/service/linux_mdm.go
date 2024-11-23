package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) LinuxHostDiskEncryptionStatus(ctx context.Context, host fleet.Host) (fleet.HostMDMDiskEncryption, error) {
	if !host.IsLUKSSupported() {
		return fleet.HostMDMDiskEncryption{}, nil
	}

	actionRequired := fleet.DiskEncryptionActionRequired
	verified := fleet.DiskEncryptionVerified
	failed := fleet.DiskEncryptionFailed

	key, err := svc.ds.GetHostDiskEncryptionKey(ctx, host.ID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return fleet.HostMDMDiskEncryption{
				Status: &actionRequired,
			}, nil
		}
		return fleet.HostMDMDiskEncryption{}, err
	}

	if key.ClientError != "" {
		return fleet.HostMDMDiskEncryption{
			Status: &failed,
			Detail: key.ClientError,
		}, nil
	}

	if key.Base64Encrypted == "" {
		return fleet.HostMDMDiskEncryption{
			Status: &actionRequired,
		}, nil
	}

	return fleet.HostMDMDiskEncryption{
		Status: &verified,
	}, nil
}

func (svc *Service) GetLinuxDiskEncryptionSummary(ctx context.Context, teamId *uint) (*fleet.MDMLinuxDiskEncryptionSummary, error) {
	if err := svc.authz.Authorize(ctx, fleet.MDMConfigProfileAuthz{TeamID: teamId}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	if svc.config.Server.PrivateKey == "" {
		return nil, ctxerr.New(ctx, "Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
	}

	ps, err := svc.ds.GetLinuxDiskEncryptionSummary(ctx, teamId)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	return &ps, nil
}
