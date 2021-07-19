package fleet

import "time"

type StatisticsPayload struct {
	AnonymousIdentifier string `json:"anonymousIdentifier"`
	FleetVersion        string `json:"fleetVersion"`
	NumHostsEnrolled    int    `json:"numHostsEnrolled"`
}

type StatisticsStore interface {
	ShouldSendStatistics(frequency time.Duration) (StatisticsPayload, bool, error)
	RecordStatisticsSent() error
}

const (
	StatisticsFrequency = time.Hour * 24 * 7
)
