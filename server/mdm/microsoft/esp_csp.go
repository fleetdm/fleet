package microsoft_mdm

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ESP item installation states as defined by the EnrollmentStatusTracking CSP.
// https://learn.microsoft.com/en-us/windows/client-management/mdm/enrollmentstatustracking-csp
const (
	ESPItemStatusNotInstalled uint = 1
	ESPItemStatusCompleted    uint = 3
	ESPItemStatusError        uint = 4
)

// ESPTimeoutSeconds is the default timeout for the Enrollment Status Page (3 hours).
const ESPTimeoutSeconds = 3 * 60 * 60

// ESPProfileTrackingInfo holds the information needed to track a profile on the ESP.
type ESPProfileTrackingInfo struct {
	// ProfileUUID is the Fleet profile UUID, used to build unique LocURIs.
	ProfileUUID string
	// TopLocURI is the LocURI of the first setting in the profile, used for ExpectedPolicies.
	TopLocURI string
	// IsSCEP indicates this is a SCEP certificate profile. SCEP profiles are
	// tracked under Certificates (ExpectedSCEPCerts) on the ESP, not under
	// Security policies (ExpectedPolicies/TrackingPolicies). Fleet's profile
	// validator enforces that SCEP profiles cannot contain non-SCEP settings,
	// so a profile is either entirely SCEP or has no SCEP at all.
	IsSCEP bool
}

// ESPSoftwareTrackingInfo holds the information needed to track a software item on the ESP.
type ESPSoftwareTrackingInfo struct {
	// Name is a display-friendly name for the software item, used in the tracking path.
	Name string
	// Status is the current installation status (ESPItemStatus* constant).
	Status uint
}

// SetupExperienceStatusToESP converts a Fleet setup experience status to an ESP item status.
func SetupExperienceStatusToESP(status fleet.SetupExperienceStatusResultStatus) uint {
	switch status {
	case fleet.SetupExperienceStatusSuccess:
		return ESPItemStatusCompleted
	case fleet.SetupExperienceStatusFailure:
		return ESPItemStatusError
	case fleet.SetupExperienceStatusPending, fleet.SetupExperienceStatusRunning:
		return ESPItemStatusNotInstalled
	default:
		return ESPItemStatusNotInstalled
	}
}
