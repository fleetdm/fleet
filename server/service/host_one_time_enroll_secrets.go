package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
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
// be a shared one) or was minted for a platform whose switch is off.
func (svc *Service) lookupOneTimeEnrollSecret(ctx context.Context, secret string) (*fleet.HostOneTimeEnrollSecret, error) {
	oneTime, err := svc.ds.GetHostOneTimeEnrollSecret(ctx, secret)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	// The stored platform, not the presented one.
	if !svc.config.Auth.OneTimeEnrollSecretsEnabledForPlatform(oneTime.Platform) {
		return nil, nil
	}
	return oneTime, nil
}

// recordEnrollmentRejected logs a refused enrollment and records a
// host_enrollment_rejected activity, at most once per host and reason within
// enrollmentRejectedActivityTTL. It never fails the caller.
func (svc *Service) recordEnrollmentRejected(ctx context.Context, reason string, hostID *uint, attempt enrollmentAttempt) {
	logArgs := []any{"reason", reason, "enrollment_plane", attempt.plane}
	if hostID != nil {
		logArgs = append(logArgs, "host_id", *hostID)
	}
	logArgs = append(logArgs,
		"platform", attempt.platform,
		"hardware_uuid", attempt.hardwareUUID,
		"hardware_serial", attempt.hardwareSerial,
	)
	svc.logger.WarnContext(ctx, "enrollment rejected", logArgs...)

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

// isWindowsEnrollSecretProfile reports whether the profile is the Fleet-managed Fleetd enroll secret profile, whatever the switch.
func isWindowsEnrollSecretProfile(profileUUID, profileName string) bool {
	return strings.HasPrefix(profileUUID, fleet.MDMWindowsProfileUUIDPrefix) && profileName == mdm.FleetWindowsEnrollSecretProfileName
}

// deliversOneTimeEnrollSecret reports whether resending this profile mints a new enrollment credential for the host: the fleetd
// configuration profile on Apple, the enroll secret profile on Windows. Both names are reserved, so users cannot upload a
// profile that impersonates one, which is what makes the name a reliable discriminator.
func deliversOneTimeEnrollSecret(auth config.AuthConfig, profileUUID, profileName string) bool {
	return (auth.UseOneTimeEnrollSecrets && isFleetdConfigProfile(profileUUID, profileName)) ||
		(auth.MDMWindowsOneTimeEnrollSecrets && isWindowsEnrollSecretProfile(profileUUID, profileName))
}

// rejectSharedSecretForWindowsMDMHosts reports whether a shared enroll secret must be refused for a Windows host enrolled in Fleet MDM.
func rejectSharedSecretForWindowsMDMHosts(auth config.AuthConfig, appConfig *fleet.AppConfig) bool {
	return auth.MDMWindowsOneTimeEnrollSecrets && appConfig.MDM.WindowsEnabledAndConfigured
}

// errWindowsEnrollSecretProfileOff refuses a resend of the Fleetd enroll secret profile while auth.mdm_windows_one_time_enroll_secrets is off.
func errWindowsEnrollSecretProfileOff() error {
	return fleet.NewInvalidArgumentError("HostMDMProfile",
		"Couldn’t resend. The "+mdm.FleetWindowsEnrollSecretProfileName+
			" profile is only used when one-time enroll secrets are enabled for Windows.").WithStatus(http.StatusConflict)
}

// expandWindowsHostSecrets expands host-scoped secrets ($FLEET_HOST_SECRET_*) in a SyncML document about to be delivered to the
// given Windows MDM enrollment. A document without any is returned untouched.
func (svc *Service) expandWindowsHostSecrets(ctx context.Context, document string, enrollmentID uint) (string, error) {
	hostSecrets := fleet.ContainsPrefixVars(document, fleet.HostSecretPrefix)
	if len(hostSecrets) == 0 {
		return document, nil
	}

	secretValues := make(map[string]string, len(hostSecrets))
	for _, secretType := range hostSecrets {
		if secretType != fleet.HostSecretEnrollSecret {
			return "", ctxerr.Errorf(ctx, "host secret type %s is not supported on Windows", secretType)
		}
		// Using primary: the fleetd install and the push to a deleted host's enrollment are minted and delivered in the same
		// management session, so a replica even slightly behind would resolve to nothing.
		secret, err := svc.ds.GetLiveWindowsMDMOneTimeEnrollSecret(ctxdb.RequirePrimary(ctx, true), enrollmentID)
		if err != nil {
			return "", ctxerr.Wrapf(ctx, err, "resolving one-time enroll secret for windows mdm enrollment %d", enrollmentID)
		}
		// Empty once the secret is consumed. The value is then delivered empty on purpose.
		secretValues[secretType] = secret
	}

	// Windows profiles and commands are always XML, so the substituted value is XML-escaped. The tokens are URL-safe base64 and
	// so never actually need it, but the escaping is what keeps that an implementation detail.
	return fleet.MaybeExpand(document, func(s string, _, _ int) (string, bool) {
		if !strings.HasPrefix(s, fleet.HostSecretPrefix) {
			return "", false
		}
		val, ok := secretValues[strings.TrimPrefix(s, fleet.HostSecretPrefix)]
		if !ok {
			return "", false
		}
		var b strings.Builder
		if err := xml.EscapeText(&b, []byte(val)); err != nil {
			return "", false
		}
		return b.String(), true
	}), nil
}

// linkWindowsEnrollmentFromOneTimeSecret links the enrolling host to the Windows MDM enrollment its one-time enroll secret was
// minted for. Failures here are logged, not returned. Linkage is post-enrollment bookkeeping, and enroll checks happen earlier:
// consumption already refused a host that another Windows MDM enrollment claims, so unlike the serial branch this needs no
// conflict check of its own.
func (svc *Service) linkWindowsEnrollmentFromOneTimeSecret(ctx context.Context, host *fleet.Host, enrollmentID uint) {
	// The primary: this runs during enrollment, moments after the rows involved were written, and a replica could miss them.
	device, err := svc.ds.MDMWindowsGetEnrolledDeviceByID(ctxdb.RequirePrimary(ctx, true), enrollmentID)
	if err != nil {
		svc.logger.ErrorContext(ctx, "failed to load windows mdm enrollment for one-time enroll secret linkage",
			"err", err, "host_uuid", host.UUID, "enrollment_id", enrollmentID)
		ctxerr.Handle(ctx, err)
		return
	}

	linked, err := osquery_utils.LinkWindowsHostMDMEnrollment(ctx, svc.logger, svc.ds, host.ID, host.UUID, device.MDMDeviceID)
	if err != nil {
		svc.logger.ErrorContext(ctx, "failed to link windows mdm enrollment from one-time enroll secret",
			"err", err, "host_uuid", host.UUID, "device_id", device.MDMDeviceID)
		ctxerr.Handle(ctx, err)
		return
	}
	if linked {
		// The enrollment predates this host record, so its mdm_enrolled activity was deferred; record it now that there is a
		// host to attribute it to, rather than waiting for the next management session.
		device.HostUUID = host.UUID
		svc.maybeCreateWindowsMDMEnrolledActivity(ctx, device)
	}
}
