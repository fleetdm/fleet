package service

import (
	"context"

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
