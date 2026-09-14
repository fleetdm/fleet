package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
)

// enrollmentRejectedActivityTTL bounds how often a host_enrollment_rejected
// activity is recorded per host and reason. A host stuck with a spent secret
// retries every few minutes forever so we don't want to spam the activity log.
const enrollmentRejectedActivityTTL = 12 * time.Hour

// enrollmentRejectedKeyPrefix is the key-value store namespace for the activity
// rate limit. The braces keep all keys in one Redis Cluster slot, like the
// other Fleet key prefixes.
const enrollmentRejectedKeyPrefix = "{enrollment_rejected}"

// enrollmentAttempt carries the identifiers an agent presented, for logging and
// the rejection activity.
type enrollmentAttempt struct {
	plane          fleet.EnrollmentPlane
	platform       string
	hardwareUUID   string
	hardwareSerial string
}

// lookupOneTimeEnrollSecret returns the one-time enroll secret matching the
// presented value, or nil when the value is not a one-time secret (it may still
// be a shared one).
func (svc *Service) lookupOneTimeEnrollSecret(ctx context.Context, secret string) (*fleet.HostOneTimeEnrollSecret, error) {
	oneTime, err := svc.ds.GetHostOneTimeEnrollSecret(ctx, secret)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return oneTime, nil
}

// recordEnrollmentRejected logs a refused enrollment and records a
// host_enrollment_rejected activity, at most once per host and reason within
// enrollmentRejectedActivityTTL. It never fails the caller.
func (svc *Service) recordEnrollmentRejected(ctx context.Context, reason string, hostID *uint, attempt enrollmentAttempt) {
	svc.logger.WarnContext(ctx, "enrollment rejected",
		"reason", reason,
		"enrollment_plane", attempt.plane,
		"host_id", hostID,
		"platform", attempt.platform,
		"hardware_uuid", attempt.hardwareUUID,
		"hardware_serial", attempt.hardwareSerial,
	)

	// Get then Set is not atomic, so two servers can race into a duplicate
	// activity; that is acceptable for a rate limit. The key is reserved only
	// after the activity is written, so a failed write is retried next time
	// rather than silenced for the TTL.
	var rateLimitKey string
	if svc.keyValueStore != nil {
		subject := attempt.hardwareUUID
		if hostID != nil {
			subject = fmt.Sprintf("host:%d", *hostID)
		}
		rateLimitKey = fmt.Sprintf("%s:%s:%s", enrollmentRejectedKeyPrefix, subject, reason)
		existing, err := svc.keyValueStore.Get(ctx, rateLimitKey)
		switch {
		case err != nil:
			svc.logger.ErrorContext(ctx, "checking enrollment rejection activity rate limit", "err", err)
		case existing != nil:
			return
		}
	}

	var displayName string
	if hostID != nil {
		if host, err := svc.ds.HostLite(ctx, *hostID); err == nil {
			displayName = host.DisplayName()
		}
	}
	if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeHostEnrollmentRejected{
		HostID:          hostID,
		HostDisplayName: displayName,
		HostSerial:      attempt.hardwareSerial,
		HostUUID:        attempt.hardwareUUID,
		Platform:        attempt.platform,
		EnrollmentPlane: string(attempt.plane),
		Reason:          reason,
	}); err != nil {
		svc.logger.ErrorContext(ctx, "record enrollment rejected activity", "err", err)
		return
	}

	if rateLimitKey != "" {
		if err := svc.keyValueStore.Set(ctx, rateLimitKey, "1", enrollmentRejectedActivityTTL); err != nil {
			svc.logger.ErrorContext(ctx, "setting enrollment rejection activity rate limit", "err", err)
		}
	}
}

// rejectedHostID returns the host a rejected enrollment targeted, preferring
// the datastore's answer and falling back to the host the presented one-time
// secret was bound to (the row can be deleted between lookup and consumption).
func rejectedHostID(rejected *fleet.EnrollmentRejectedError, oneTime *fleet.HostOneTimeEnrollSecret) *uint {
	if rejected.HostID != nil {
		return rejected.HostID
	}
	if oneTime != nil {
		return oneTime.HostID
	}
	return nil
}

// isFleetdConfigProfile reports whether the profile is the Fleet-managed fleetd
// configuration profile. Users cannot upload a profile with this name
// (MDMAppleConfigProfile.ValidateUserProvided rejects it), so the name is a
// reliable discriminator.
func isFleetdConfigProfile(profileUUID, profileName string) bool {
	return strings.HasPrefix(profileUUID, fleet.MDMAppleProfileUUIDPrefix) && profileName == mdm.FleetdConfigProfileName
}
