package fleet

import (
	"encoding/json"
	"time"
)

type StatisticsPayload struct {
	AnonymousIdentifier            string                             `json:"anonymousIdentifier"`
	FleetVersion                   string                             `json:"fleetVersion"`
	LicenseTier                    string                             `json:"licenseTier"`
	NumHostsEnrolled               int                                `json:"numHostsEnrolled"`
	NumUsers                       int                                `json:"numUsers"`
	NumTeams                       int                                `json:"numTeams"`
	NumPolicies                    int                                `json:"numPolicies"`
	NumLabels                      int                                `json:"numLabels"`
	SoftwareInventoryEnabled       bool                               `json:"softwareInventoryEnabled"`
	VulnDetectionEnabled           bool                               `json:"vulnDetectionEnabled"`
	SystemUsersEnabled             bool                               `json:"systemUsersEnabled"`
	HostsStatusWebHookEnabled      bool                               `json:"hostsStatusWebHookEnabled"`
	NumWeeklyActiveUsers           int                                `json:"numWeeklyActiveUsers"`
	HostsEnrolledByOperatingSystem map[string][]HostsCountByOSVersion `json:"hostsEnrolledByOperatingSystem"`
	StoredErrors                   json.RawMessage                    `json:"storedErrors"`
	// NumHostsNotResponding is the count of hosts that haven't submitted results for distributed queries.
	//
	// Notes:
	//   - We use `2 * interval`, because of the artificial jitter added to the intervals in Fleet.
	//   - Default values for:
	//     - host.DistributedInterval is usually 10s.
	//     - svc.config.Osquery.DetailUpdateInterval is usually 1h.
	//   - Count only includes hosts seen during the last 7 days.
	NumHostsNotResponding int `json:"numHostsNotResponding"`
}

type HostsCountByOSVersion struct {
	Version     string `json:"version"`
	NumEnrolled int    `json:"numEnrolled"`
}

const (
	StatisticsFrequency = time.Hour * 24 * 7
)
