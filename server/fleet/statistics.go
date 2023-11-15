package fleet

import (
	"encoding/json"
	"time"
)

type StatisticsPayload struct {
	AnonymousIdentifier       string `json:"anonymousIdentifier"`
	FleetVersion              string `json:"fleetVersion"`
	LicenseTier               string `json:"licenseTier"`
	Organization              string `json:"organization"`
	NumHostsEnrolled          int    `json:"numHostsEnrolled"`
	NumUsers                  int    `json:"numUsers"`
	NumTeams                  int    `json:"numTeams"`
	NumPolicies               int    `json:"numPolicies"`
	NumLabels                 int    `json:"numLabels"`
	SoftwareInventoryEnabled  bool   `json:"softwareInventoryEnabled"`
	VulnDetectionEnabled      bool   `json:"vulnDetectionEnabled"`
	SystemUsersEnabled        bool   `json:"systemUsersEnabled"`
	HostsStatusWebHookEnabled bool   `json:"hostsStatusWebHookEnabled"`
	MDMMacOsEnabled           bool   `json:"mdmMacOsEnabled"`
	HostExpiryEnabled         bool   `json:"hostExpiryEnabled"`
	MDMWindowsEnabled         bool   `json:"mdmWindowsEnabled"`
	LiveQueryDisabled         bool   `json:"liveQueryDisabled"`
	NumWeeklyActiveUsers      int    `json:"numWeeklyActiveUsers"`
	// NumWeeklyPolicyViolationDaysActual is an aggregate count of actual policy violation days. One
	// policy violation day is added for each policy that a host is failing as of the time the count
	// is incremented. The count increments once per 24-hour interval and resets each week.
	NumWeeklyPolicyViolationDaysActual int `json:"numWeeklyPolicyViolationDaysActual"`
	// NumWeeklyPolicyViolationDaysActual is an aggregate count of possible policy violation
	// days. The count is incremented by the organization's total number of policies
	// mulitplied by the total number of hosts as of the time the count is incremented. The count
	// increments once per 24-hour interval and resets each week.
	NumWeeklyPolicyViolationDaysPossible int                                `json:"numWeeklyPolicyViolationDaysPossible"`
	HostsEnrolledByOperatingSystem       map[string][]HostsCountByOSVersion `json:"hostsEnrolledByOperatingSystem"`
	// HostsEnrolledByOrbitVersion is a count of hosts enrolled to Fleet grouped by orbit version
	HostsEnrolledByOrbitVersion []HostsCountByOrbitVersion `json:"hostsEnrolledByOrbitVersion"`
	// HostsEnrolledByOsqueryVersion is a count of hosts enrolled to Fleet grouped by osquery version
	HostsEnrolledByOsqueryVersion []HostsCountByOsqueryVersion `json:"hostsEnrolledByOsqueryVersion"`
	StoredErrors                  json.RawMessage              `json:"storedErrors"`
	// NumHostsNotResponding is a count of hosts that connect to Fleet successfully but fail to submit results for distributed queries.
	NumHostsNotResponding int `json:"numHostsNotResponding"`
}

type HostsCountByOrbitVersion struct {
	OrbitVersion string `json:"orbitVersion" db:"orbit_version"`
	NumHosts     int    `json:"numHosts" db:"num_hosts"`
}
type HostsCountByOsqueryVersion struct {
	OsqueryVersion string `json:"osqueryVersion" db:"osquery_version"`
	NumHosts       int    `json:"numHosts" db:"num_hosts"`
}

type HostsCountByOSVersion struct {
	Version     string `json:"version"`
	NumEnrolled int    `json:"numEnrolled"`
}

const (
	StatisticsFrequency = time.Hour * 24
)
