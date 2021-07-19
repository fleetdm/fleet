package fleet

type StatisticsPayload struct {
	AnonymousIdentifier string `json:"anonymousIdentifier"`
	FleetVersion        string `json:"fleetVersion"`
	NumHostsEnrolled    int    `json:"numHostsEnrolled"`
}

type StatisticsStore interface {
	ShouldSendStatistics() (StatisticsPayload, bool, error)
	RecordStatisticsSent() error
}
