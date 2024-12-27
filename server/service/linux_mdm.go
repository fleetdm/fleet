package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) LinuxHostDiskEncryptionStatus(ctx context.Context, host fleet.Host) (fleet.HostMDMDiskEncryption, error) {
	fmt.Printf("\n\nINSIDE LinuxHostDiskEncryptionStatus\n\n")
	if !host.IsLUKSSupported() {
		fmt.Printf("\n\nHost LUKS NOT SUPPORTED, RETURNING EMPTY STRUCT\n\n")
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

func (svc *Service) GetMDMLinuxProfilesSummary(ctx context.Context, teamId *uint) (summary fleet.MDMProfilesSummary, err error) {
	if err = svc.authz.Authorize(ctx, fleet.MDMConfigProfileAuthz{TeamID: teamId}, fleet.ActionRead); err != nil {
		return summary, ctxerr.Wrap(ctx, err)
	}

	// Linux doesn't have configuration profiles, so if we aren't enforcing disk encryption we have nothing to report
	includeDiskEncryptionStats, err := svc.ds.GetConfigEnableDiskEncryption(ctx, teamId)
	if err != nil {
		return summary, ctxerr.Wrap(ctx, err)
	} else if !includeDiskEncryptionStats {
		return summary, nil
	}

	counts, err := svc.ds.GetLinuxDiskEncryptionSummary(ctx, teamId)
	if err != nil {
		return summary, ctxerr.Wrap(ctx, err)
	}

	return fleet.MDMProfilesSummary{
		Verified: counts.Verified,
		Pending:  counts.ActionRequired,
		Failed:   counts.Failed,
	}, nil
}
