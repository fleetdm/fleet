package fleet

import (
	"fmt"
	"strings"
	"time"
)

// HostOneTimeEnrollSecretSecondPlaneWindow bounds how long after the first
// use of a one-time enroll secret the other enrollment plane may still use it.
// Orbit enrolls first and hands the same secret to osqueryd, which normally
// enrolls seconds later, but a server outage right after orbit's enroll can
// stretch osquery's retry backoff to about 10 minutes.
const HostOneTimeEnrollSecretSecondPlaneWindow = 60 * time.Minute

// EnrollmentPlane identifies which fleetd component is enrolling.
type EnrollmentPlane string

const (
	EnrollmentPlaneOrbit   EnrollmentPlane = "orbit"
	EnrollmentPlaneOsquery EnrollmentPlane = "osquery"
)

// HostOneTimeEnrollSecret is a per-device, single-use enroll secret minted when an MDM-enrolled host is handed the credential it
// will enroll with: an Apple host fetching its fleetd configuration profile, or a Windows host being sent the fleetd installer.
// It may be used once per enrollment plane.
//
// What it binds to differs by platform, because Apple has a host to bind to and Windows does not yet. See MatchesHost.
type HostOneTimeEnrollSecret struct {
	ID     uint   `db:"id"`
	Secret string `db:"secret"`
	HostID *uint  `db:"host_id"`
	// MDMWindowsEnrollmentID binds the secret to a Windows MDM enrollment instead of to a host. Windows mints before a hosts row
	// exists, because the automatic enrollment flows carry no Fleet host UUID, so HostID stays nil until an agent enrolls with
	// the secret. Nil on the Apple path, which always has a host to bind to.
	MDMWindowsEnrollmentID *uint      `db:"mdm_windows_enrollment_id"`
	TeamID                 *uint      `db:"team_id"`
	Platform               string     `db:"platform"`
	HardwareUUID           string     `db:"hardware_uuid"`
	HardwareSerial         string     `db:"hardware_serial"`
	CreatedAt              time.Time  `db:"created_at"`
	ConsumedAt             *time.Time `db:"consumed_at"`
	OrbitUsedAt            *time.Time `db:"orbit_used_at"`
	OsqueryUsedAt          *time.Time `db:"osquery_used_at"`
}

// MatchesHost reports whether the identifiers presented by an enrolling agent
// match the binding captured when the secret was minted. Comparison is
// case-insensitive because MDM and osquery do not guarantee the same casing.
func (s *HostOneTimeEnrollSecret) MatchesHost(platform, hardwareUUID, hardwareSerial string) bool {
	if s.IsMDMEnrollmentBound() {
		return s.matchesCapturedIdentifiers(platform, hardwareUUID, hardwareSerial)
	}
	return strings.EqualFold(s.Platform, platform) &&
		strings.EqualFold(s.HardwareUUID, hardwareUUID) &&
		strings.EqualFold(s.HardwareSerial, hardwareSerial)
}

// IsMDMEnrollmentBound reports whether the secret was minted for an MDM enrollment rather than for a host. Only Windows mints
// this way, because its automatic enrollment flows have no host to bind to yet.
func (s *HostOneTimeEnrollSecret) IsMDMEnrollmentBound() bool {
	return s.MDMWindowsEnrollmentID != nil
}

// matchesCapturedIdentifiers compares only the identifiers both sides actually have. An enrollment-bound secret is minted before
// the device has reported most of them: the hardware UUID is never known at that point, and the serial only if a DevDetail
// response already landed. The agent is equally partial in the other direction, and on Windows always is: OrbitHostInfo
// documents HardwareSerial as set on macOS and Linux only, so it arrives empty from every Windows host. Requiring a captured
// serial to equal a presented one therefore rejects exactly the hosts furthest along in enrollment, the ones whose DevDetail
// landed before the secret was minted. An empty value on either side means "nothing to compare" and is skipped.
//
// What makes that safe is that the binding is the enrollment itself: the secret was delivered only over that device's own MDM
// channel, and possession is the credential. The identifiers are provenance, recorded on first use.
func (s *HostOneTimeEnrollSecret) matchesCapturedIdentifiers(platform, hardwareUUID, hardwareSerial string) bool {
	captured := func(stored, presented string) bool {
		return stored == "" || presented == "" || strings.EqualFold(stored, presented)
	}
	return captured(s.Platform, platform) &&
		captured(s.HardwareUUID, hardwareUUID) &&
		captured(s.HardwareSerial, hardwareSerial)
}

// UsedAt returns the time the secret was used by the given plane, or nil.
func (s *HostOneTimeEnrollSecret) UsedAt(plane EnrollmentPlane) *time.Time {
	switch plane {
	case EnrollmentPlaneOrbit:
		return s.OrbitUsedAt
	case EnrollmentPlaneOsquery:
		return s.OsqueryUsedAt
	}
	return nil
}

// Enrollment rejection reasons, recorded in the host_enrollment_rejected activity.
const (
	// EnrollmentRejectedOneTimeSecretSpent: a one-time secret was presented by
	// its bound host after it had already been used by that plane, or outside
	// the second-plane window. The host needs an admin to resend the fleetd
	// configuration profile.
	EnrollmentRejectedOneTimeSecretSpent = "one_time_secret_spent"
	// EnrollmentRejectedOneTimeSecretIdentifierMismatch: a one-time secret was
	// presented with identifiers that do not match the device it was minted for.
	EnrollmentRejectedOneTimeSecretIdentifierMismatch = "one_time_secret_identifier_mismatch"
	// EnrollmentRejectedSharedSecretForMDMManagedHost: a shared (team or
	// global) enroll secret was used to claim an Apple host that is enrolled in
	// Fleet MDM or assigned to Fleet in Apple Business.
	EnrollmentRejectedSharedSecretForMDMManagedHost = "shared_secret_for_mdm_managed_host"
)

// EnrollmentRejectedError is returned by the datastore enroll methods when an
// enrollment is refused by the one-time enroll secret rules. HostID is the host
// the attempt targeted, when known.
type EnrollmentRejectedError struct {
	Reason string
	HostID *uint
}

func (e *EnrollmentRejectedError) Error() string {
	if e.HostID != nil {
		return fmt.Sprintf("enrollment rejected: %s (host %d)", e.Reason, *e.HostID)
	}
	return fmt.Sprintf("enrollment rejected: %s", e.Reason)
}
