package android

type NotificationType string

const (
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
}
