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
// will enroll with: an Apple host fetching its fleetd configuration profile, or a Windows host being sent the fleetd installer or
// having its Fleetd enroll secret profile resent by an administrator. It may be used once per enrollment plane.
type HostOneTimeEnrollSecret struct {
	ID     uint   `db:"id"`
	Secret string `db:"secret"`
	HostID *uint  `db:"host_id"`
	// MDMWindowsEnrollmentID binds the secret to a Windows MDM enrollment instead of to a host.
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

// WindowsEnrollmentID returns the Windows MDM enrollment the secret was minted for, or nil when there is none.
func (s *HostOneTimeEnrollSecret) WindowsEnrollmentID() *uint {
	if s == nil {
		return nil
	}
	return s.MDMWindowsEnrollmentID
}

// matchesCapturedIdentifiers compares only the identifiers both sides actually have. An enrollment-bound secret is minted before
// the device has reported most of them: the hardware UUID is never known at that point, and the serial only if a DevDetail
// response already landed. The agent is equally partial in the other direction. An empty value on either side means "nothing to
// compare" and is skipped.
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
	// Fleet MDM or assigned to Fleet in Apple Business, or a Windows host that is
	// enrolled in Fleet MDM.
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
