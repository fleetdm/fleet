package android

type NotificationType string

const (
	PubSubTest         NotificationType = "test"
	PubSubEnrollment   NotificationType = "ENROLLMENT"
	PubSubStatusReport NotificationType = "STATUS_REPORT"
	PubSubCommand      NotificationType = "COMMAND"
	PubSubUsageLogs    NotificationType = "USAGE_LOGS"
)

type DeviceState string

const (
	DeviceStateDeleted DeviceState = "DELETED"
)

type PubSubMessage struct {
	Attributes map[string]string `json:"attributes"`
	Data       string            `json:"data"`
	// MessageID and PublishTime are set by Google Pub/Sub on the push envelope as
	// siblings of Attributes/Data. MessageID is stable across at-least-once
	// redeliveries of the same message; PublishTime is an RFC3339 timestamp used as
	// a staleness fallback when the AMAPI payload carries no event timestamp.
	MessageID   string `json:"messageId"`
	PublishTime string `json:"publishTime"`
}
