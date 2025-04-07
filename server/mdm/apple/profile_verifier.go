package apple_mdm

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
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
// - When host details are reported via osquery, the Fleet server ingests a list of installed
//   profiles and compares the reported profiles with the list of profiles expected to be
//   installed on the host. Expected profiles comprise the profiles that belong to the host's assigned
//   team (or no team, as applicable). If an expected profile is found, the verification status is
//   set to "verified". If an expected profile is missing from the reported results, the server determines
//   if the profile should be retried (in which case, a new install profile command will be enqueued by the server)
//   or marked as "failed" and updates the datastore accordingly.

// VerifyHostMDMProfiles performs the verification of the MDM profiles installed on a host and
// updates the verification status in the datastore. It is intended to be called by Fleet osquery
// service when the Fleet server ingests host details.
func VerifyHostMDMProfiles(ctx context.Context, ds fleet.ProfileVerificationStore, host *fleet.Host, installed map[string]*fleet.HostMacOSProfile) error {
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
		counts, err := ds.GetHostMDMProfilesRetryCounts(ctx, host)
		if err != nil {
			return err
		}
		retriesByProfileIdentifier := make(map[string]uint, len(counts))
		for _, r := range counts {
			retriesByProfileIdentifier[r.ProfileIdentifier] = r.Retries
		}
		for _, key := range missing {
			if retriesByProfileIdentifier[key] < mdm.MaxProfileRetries {
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

	return ds.UpdateHostMDMProfilesVerification(ctx, host, verified, toFail, toRetry)
}

// HandleHostMDMProfileInstallResult ingests the result of an install profile command reported via
// the MDM protocol and updates the verification status in the datastore. It is intended to be
// called by the Fleet MDM checkin and command service install profile request handler.
func HandleHostMDMProfileInstallResult(ctx context.Context, ds fleet.ProfileVerificationStore, hostUUID string, cmdUUID string, status *fleet.MDMDeliveryStatus, detail string) error {
	if status != nil && *status == fleet.MDMDeliveryFailed {
		// Here we set the host.Platform to "darwin" but it applies to iOS/iPadOS too.
		// The logic in GetHostMDMProfileRetryCountByCommandUUID and UpdateHostMDMProfilesVerification
		// is the exact same when platform is "darwin", "ios" or "ipados".
		host := &fleet.Host{UUID: hostUUID, Platform: "darwin"}
		m, err := ds.GetHostMDMProfileRetryCountByCommandUUID(ctx, host, cmdUUID)
		if err != nil {
			return err
		}

		if m.Retries < mdm.MaxProfileRetries {
			// if we haven't hit the max retries, we set the host profile status to nil (which
			// causes an install profile command to be enqueued the next time the profile
			// manager cron runs) and increment the retry count
			return ds.UpdateHostMDMProfilesVerification(ctx, host, nil, nil, []string{m.ProfileIdentifier})
		}
	}

	// otherwise update status and detail as usual
	err := ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		CommandUUID:   cmdUUID,
		HostUUID:      hostUUID,
		Status:        status,
		Detail:        detail,
		OperationType: fleet.MDMOperationTypeInstall,
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating host MDM Apple profile install result")
	}
	return nil
}
