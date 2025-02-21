package android

const (
	PubSubEnrollment   = "ENROLLMENT"
	PubSubStatusReport = "STATUS_REPORT"
	PubSubCommand      = "COMMAND"
	PubSubUsageLogs    = "USAGE_LOGS"
)

type PubSubMessage struct {
	Attributes map[string]string `json:"attributes"`
	Data       string            `json:"data"`
}
