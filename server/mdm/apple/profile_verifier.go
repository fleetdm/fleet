package apple_mdm

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ProfileVerificationStore is the minimal interface required to get and update the verification
// status of a host's MDM profiles. The Fleet Datastore satisfies this interface.
type ProfileVerificationStore interface {
	CheckHostMDMProfileCommandWithinGracePeriod(ctx context.Context, hostUUID string, period time.Duration) (bool, error)
	GetHostMDMProfilesExpectedForVerification(ctx context.Context, host *fleet.Host) (map[string]*fleet.ExpectedMDMProfile, error)
	UpdateHostMDMProfilesVerification(ctx context.Context, host *fleet.Host, verified, failed []string) error
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
	hostWithinGracePeriod, err := ds.CheckHostMDMProfileCommandWithinGracePeriod(ctx, host.UUID, 1*time.Hour)
	if err != nil {
		return err
	}

	failed := make([]string, 0, len(expected))
	verified := make([]string, 0, len(expected))
	for key, ep := range expected {
		withinGracePeriod := hostWithinGracePeriod || ep.IsWithinGracePeriod(host.DetailUpdatedAt)
		ip, ok := installed[key]
		if !ok {
			// expected profile is missing from host
			if !withinGracePeriod {
				failed = append(failed, key)
			}
			continue
		}
		if ip.InstallDate.Before(ep.EarliestInstallDate) {
			// installed profile is outdated
			if !withinGracePeriod {
				failed = append(failed, key)
			}
			continue
		}
		verified = append(verified, key)
	}

	return ds.UpdateHostMDMProfilesVerification(ctx, host, verified, failed)
}
