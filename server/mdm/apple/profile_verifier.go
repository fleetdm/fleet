package apple_mdm

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ProfileVerificationStore is the minimal interface required to get and update the verification
// status of a host's MDM profiles. The Fleet Datastore satisfies this interface.
type ProfileVerificationStore interface {
	GetHostMDMProfilesExpectedForVerification(ctx context.Context, host *fleet.Host) (map[string]*fleet.ExpectedMDMProfile, error)
	GetHostMDMProfilesRetryCounts(ctx context.Context, hostUUID string) ([]fleet.HostMDMProfileRetryCount, error)
	GetHostMDMProfileRetryCountByCommandUUID(ctx context.Context, hostUUID, commandUUID string) (fleet.HostMDMProfileRetryCount, error)
	UpdateHostMDMProfilesVerification(ctx context.Context, hostUUID string, toVerify, toFail, toRetry []string) error
	UpdateOrDeleteHostMDMAppleProfile(ctx context.Context, profile *fleet.HostMDMAppleProfile) error
}

var _ ProfileVerificationStore = (fleet.Datastore)(nil)

const maxRetries = 1

// VerifyHostMDMProfiles performs the verification of the MDM profiles installed on a host and
// updates the verification status in the datastore. It is intended to be called by Fleet osquery
// service when the Fleet server ingests host details.
func VerifyHostMDMProfiles(ctx context.Context, ds ProfileVerificationStore, host *fleet.Host, installed map[string]*fleet.HostMacOSProfile) error {
	expected, err := ds.GetHostMDMProfilesExpectedForVerification(ctx, host)
	if err != nil {
		return err
	}

	missing := make([]string, 0, len(expected))
	verified := make([]string, 0, len(expected))
	for key, ep := range expected {
		withinGracePeriod := ep.IsWithinGracePeriod(host.DetailUpdatedAt)
		ip, ok := installed[key]
		if !ok {
			// expected profile is missing from host
			if !withinGracePeriod {
				missing = append(missing, key)
			}
			continue
		}
		if ip.InstallDate.Before(ep.EarliestInstallDate) {
			// installed profile is outdated
			if !withinGracePeriod {
				missing = append(missing, key)
			}
			continue
		}
		verified = append(verified, key)
	}

	toFail := make([]string, 0, len(missing))
	toRetry := make([]string, 0, len(missing))
	if len(missing) > 0 {
		prevRetries, err := ds.GetHostMDMProfilesRetryCounts(ctx, host.UUID)
		if err != nil {
			return err
		}
		retriesByProfIdent := make(map[string]uint, len(prevRetries))
		for _, r := range prevRetries {
			retriesByProfIdent[r.ProfileIdentifier] = r.Retries
		}
		for _, key := range missing {
			if retriesByProfIdent[key] >= maxRetries {
				toFail = append(toFail, key)
			} else {
				toRetry = append(toRetry, key)
			}
		}
	}

	return ds.UpdateHostMDMProfilesVerification(ctx, host.UUID, verified, toFail, toRetry)
}

func HandleHostMDMProfileInstallResult(ctx context.Context, ds ProfileVerificationStore, hostUUID string, cmdUUID string, status *fleet.MDMAppleDeliveryStatus, detail string) error {
	if status != nil && *status == fleet.MDMAppleDeliveryFailed {
		m, err := ds.GetHostMDMProfileRetryCountByCommandUUID(ctx, hostUUID, cmdUUID)
		if err != nil {
			return err
		}

		if m.Retries < maxRetries {
			// if we haven't hit the max retries, we set the host profile status to nil (which causes an install profile command to be enqueued the next time the profile
			// manager cron runs) and increment the retry count
			return ds.UpdateHostMDMProfilesVerification(ctx, hostUUID, nil, nil, []string{m.ProfileIdentifier})
		}
	}

	// otherwise update status and detail as usual
	return ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		CommandUUID:   cmdUUID,
		HostUUID:      hostUUID,
		Status:        status,
		Detail:        detail,
		OperationType: fleet.MDMAppleOperationTypeInstall,
	})
}
