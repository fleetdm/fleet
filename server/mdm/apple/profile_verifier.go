package apple_mdm

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Profile verification is a set of related processes that run on the Fleet server to ensure that
// the MDM profiles installed on a host are the ones expected by the Fleet server. Expected profiles
// comprise the profiles that belong to the host's assigned team (or no
// team, as applicable).
//
// The Fleet server enqueues commands to install profiles on hosts via the MDM
// protocol. The Fleet server periodically runs a cron that enqueues install profile
// commands for host profiles that do not have a verification status (i.e. status is null).
// Install profile commands may be enqueued as a result of a variety of events, such as when a host
// enrolls in Fleet, when a host's team membership changes, when a new profile is uploaded, when an
// existing profile is modified, or when a failed profile is retried.
//
// Verification status of a host profile can change in the following ways:
//
// - When an install profile command is enqueued by the server, the verification status is set to "pending".
//
// - When the results of an install profile command are reported via the MDM protocol, the Fleet server
//   parses the results and updates the host's verification status for the applicable profile. If the
//   command was acknowledged, the verification status is set to "verifying". If the command resulted
//   in an error, the server determines if the profile should be retried (in which case, a new install profile
//   command will be enqueued by the server) or marked as "failed" and updates the datastore accordingly.
//
// - When host details are reported via osqueery, the Fleet server ingests a list of installed
//   profiles and compares the reported profiles with the list of profiles expected to be
//   installed on the host. Expected profiles comprise the profiles that belong to the host's assigned
//   team (or no  team, as applicable). If an expected profile is found, the verification status is
//   set to "verified". If an expected profile is missing from the reported results, the server determines
//   if the profile should be retried (in which case, a new install profile command will be enqueued by the server)
//   or marked as "failed" and updates the datastore accordingly.

// maxRetries is the maximum times an install profile command may be retried, after which marked as failed and no further
// attempts will be made to install the profile.
const maxRetries = 1

// ProfileVerificationStore is the minimal interface required to get and update the verification
// status of a host's MDM profiles. The Fleet Datastore satisfies this interface.
type ProfileVerificationStore interface {
	// GetHostMDMProfilesExpectedForVerification returns the expected MDM profiles for a given host. The map is
	// keyed by the profile identifier.
	GetHostMDMProfilesExpectedForVerification(ctx context.Context, host *fleet.Host) (map[string]*fleet.ExpectedMDMProfile, error)
	// GetHostMDMProfilesRetryCounts returns the retry counts for the specified host.
	GetHostMDMProfilesRetryCounts(ctx context.Context, hostUUID string) ([]fleet.HostMDMProfileRetryCount, error)
	// GetHostMDMProfileRetryCountByCommandUUID returns the retry count for the specified
	// command UUID and host UUID.
	GetHostMDMProfileRetryCountByCommandUUID(ctx context.Context, hostUUID, commandUUID string) (fleet.HostMDMProfileRetryCount, error)
	// UpdateVerificationHostMacOSProfiles updates status of macOS profiles installed on a given
	// host. The toVerify, toFail, and toRetry slices contain the identifiers of the profiles that
	// should be verified, failed, and retried, respectively. For each profile in the toRetry slice,
	// the retries count is incremented by 1 and the status is set to null so that an install
	// profile command is enqueued the next time the profile manager cron runs.
	UpdateHostMDMProfilesVerification(ctx context.Context, hostUUID string, toVerify, toFail, toRetry []string) error
	// UpdateOrDeleteHostMDMAppleProfile updates information about a single
	// profile status. It deletes the row if the profile operation is "remove"
	// and the status is "verifying" (i.e. successfully removed).
	UpdateOrDeleteHostMDMAppleProfile(ctx context.Context, profile *fleet.HostMDMAppleProfile) error
}

var _ ProfileVerificationStore = (fleet.Datastore)(nil)

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
		counts, err := ds.GetHostMDMProfilesRetryCounts(ctx, host.UUID)
		if err != nil {
			return err
		}
		retriesByProfileIdentifier := make(map[string]uint, len(counts))
		for _, r := range counts {
			retriesByProfileIdentifier[r.ProfileIdentifier] = r.Retries
		}
		for _, key := range missing {
			if retriesByProfileIdentifier[key] < maxRetries {
				// if we haven't hit the max retries, we set the host profile status to nil (which
				// causes an install profile command to be enqueued the next time the profile
				// manager cron runs) and increment the retry count
				toRetry = append(toRetry, key)
			} else {
				// otherwise we set the host profile status to failed
				toFail = append(toFail, key)
			}
		}
	}

	return ds.UpdateHostMDMProfilesVerification(ctx, host.UUID, verified, toFail, toRetry)
}

// HandleHostMDMProfileInstallResult ingests the result of an install profile command reported via
// the MDM protocol and updates the verification status in the datastore. It is intended to be
// called by the Fleet MDM checkin and command service install profile request handler.
func HandleHostMDMProfileInstallResult(ctx context.Context, ds ProfileVerificationStore, hostUUID string, cmdUUID string, status *fleet.MDMAppleDeliveryStatus, detail string) error {
	if status != nil && *status == fleet.MDMAppleDeliveryFailed {
		m, err := ds.GetHostMDMProfileRetryCountByCommandUUID(ctx, hostUUID, cmdUUID)
		if err != nil {
			return err
		}

		if m.Retries < maxRetries {
			// if we haven't hit the max retries, we set the host profile status to nil (which
			// causes an install profile command to be enqueued the next time the profile
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
