package fleet

import "time"

type StatisticsPayload struct {
	AnonymousIdentifier string `json:"anonymousIdentifier"`
	FleetVersion        string `json:"fleetVersion"`
	NumHostsEnrolled    int    `json:"numHostsEnrolled"`
}

const (
	StatisticsFrequency = time.Hour * 24 * 7
)
