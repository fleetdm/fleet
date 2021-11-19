package fleet

import "time"

type StatisticsPayload struct {
	AnonymousIdentifier       string `json:"anonymousIdentifier"`
	FleetVersion              string `json:"fleetVersion"`
	LicenseTier               string `json:"licenseTier"`
	NumHostsEnrolled          int    `json:"numHostsEnrolled"`
	NumUsers                  int    `json:"numUsers"`
	NumTeams                  int    `json:"numTeams"`
	NumPolicies               int    `json:"numPolicies"`
	SoftwareInventoryEnabled  bool   `json:"softwareInventoryEnabled"`
	VulnDetectionEnabled      bool   `json:"vulnDetectionEnabled"`
	SystemUsersEnabled        bool   `json:"systemUsersEnabled"`
	HostsStatusWebHookEnabled bool   `json:"hostsStatusWebHookEnabled"`
}

const (
	StatisticsFrequency = time.Hour * 24 * 7
)
