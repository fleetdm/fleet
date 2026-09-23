package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
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

// deliversOneTimeEnrollSecret reports whether resending this profile mints a new enrollment credential for the host: the fleetd
// configuration profile on Apple, the enroll secret profile on Windows. Both names are reserved, so users cannot upload a
// profile that impersonates one, which is what makes the name a reliable discriminator.
//
// Each platform is judged by its own switch, because they roll out independently. A profile whose platform is switched off
// carries no secret to protect, and the ordinary resend rules apply to it.
//
// This is deliberately broader than isFleetdConfigProfile, which stays Apple-only because it also guards the resend-from-
// verifying carve-out. That carve-out exists because an Apple profile is verified by osquery and can sit in verifying forever
// when osquery is the broken part. A Windows profile reaches verified on the SyncML ack, so it never needs it.
func deliversOneTimeEnrollSecret(auth config.AuthConfig, profileUUID, profileName string) bool {
	switch {
	case strings.HasPrefix(profileUUID, fleet.MDMAppleProfileUUIDPrefix):
		return auth.UseOneTimeEnrollSecrets && profileName == mdm.FleetdConfigProfileName
	case strings.HasPrefix(profileUUID, fleet.MDMWindowsProfileUUIDPrefix):
		return auth.MDMWindowsOneTimeEnrollSecrets && profileName == mdm.FleetWindowsEnrollSecretProfileName
	}
	return false
}

// isWindowsEnrollSecretProfile reports whether the profile is the Fleet-managed Fleetd enroll secret profile, whatever the switch.
func isWindowsEnrollSecretProfile(profileUUID, profileName string) bool {
	return strings.HasPrefix(profileUUID, fleet.MDMWindowsProfileUUIDPrefix) && profileName == mdm.FleetWindowsEnrollSecretProfileName
}

// errWindowsEnrollSecretProfileOff refuses a resend of the Fleetd enroll secret profile while
// auth.mdm_windows_one_time_enroll_secrets is off. The reconciler deletes the profile then, but a resend could still find it in the
// window before that, or after a delete that failed. It has to be refused here: the datastore mints on the profile's name because
// it cannot see the switch, so letting the resend through would hand out a credential for a feature that is disabled.
func errWindowsEnrollSecretProfileOff() error {
	return fleet.NewInvalidArgumentError("HostMDMProfile",
		"Couldn’t resend. The "+mdm.FleetWindowsEnrollSecretProfileName+
			" profile is only used when one-time enroll secrets are enabled for Windows.").WithStatus(http.StatusConflict)
}

// oneTimeWindowsEnrollmentID returns the Windows MDM enrollment a presented one-time enroll secret was minted for, or nil when
// there is none: a shared secret is not a one-time secret at all, and an Apple one-time secret binds to a host instead.
func oneTimeWindowsEnrollmentID(oneTime *fleet.HostOneTimeEnrollSecret) *uint {
	if oneTime == nil {
		return nil
	}
	return oneTime.MDMWindowsEnrollmentID
}

// linkWindowsEnrollmentFromOneTimeSecret links the enrolling host to the Windows MDM enrollment its one-time enroll secret was
// minted for.
//
// This is the stronger half of the linkage story. The paths it displaces infer the enrollment from a hardware serial that the
// device asserted about itself and that nothing corroborates, which is why they need the conflicting-hardware guard. A one-time
// secret is server-minted and was delivered only over that enrollment's own MDM channel, so presenting it identifies the
// enrollment rather than suggesting it.
//
// The conflicting-hardware guard is kept anyway. It answers a different question than the secret does: the secret proves which
// enrollment it came from, not that the host row it is landing on belongs to the same physical machine.
//
// Failures here are logged, not returned. Linkage is post-enrollment bookkeeping, and refusing the enrollment over it would
// leave a host that cannot run fleetd at all rather than one that is merely unlinked.
func (svc *Service) linkWindowsEnrollmentFromOneTimeSecret(ctx context.Context, host *fleet.Host, enrollmentID uint) {
	device, err := svc.ds.MDMWindowsGetEnrolledDeviceByID(ctx, enrollmentID)
	if err != nil {
		svc.logger.ErrorContext(ctx, "failed to load windows mdm enrollment for one-time enroll secret linkage",
			"err", err, "host_uuid", host.UUID, "enrollment_id", enrollmentID)
		return
	}

	conflicted, conflictingHardwareID, err := svc.ds.MDMWindowsConflictingEnrollmentHardwareID(ctx, host.UUID, device.MDMHardwareID)
	switch {
	case err != nil:
		svc.logger.ErrorContext(ctx, "failed to check for conflicting windows mdm enrollment during one-time secret linkage",
			"err", err, "host_uuid", host.UUID, "device_id", device.MDMDeviceID)
		return
	case conflicted:
		svc.logger.WarnContext(ctx, "refusing to link windows mdm enrollment to a host already claimed by other hardware",
			"host_uuid", host.UUID, "device_id", device.MDMDeviceID, "claimed_by_hardware_id", conflictingHardwareID)
		return
	}

	linked, err := osquery_utils.LinkWindowsHostMDMEnrollment(ctx, svc.logger, svc.ds, host.ID, host.UUID, device.MDMDeviceID)
	if err != nil {
		svc.logger.ErrorContext(ctx, "failed to link windows mdm enrollment from one-time enroll secret",
			"err", err, "host_uuid", host.UUID, "device_id", device.MDMDeviceID)
		return
	}
	if linked {
		// The enrollment predates this host record, so its mdm_enrolled activity was deferred; record it now that there is a
		// host to attribute it to, rather than waiting for the next management session.
		device.HostUUID = host.UUID
		svc.maybeCreateWindowsMDMEnrolledActivity(ctx, device)
	}
}
