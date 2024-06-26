package microwebhook

import "time"

type Event struct {
	Topic     string    `json:"topic"`
	EventID   string    `json:"event_id"`
	CreatedAt time.Time `json:"created_at"`

	AcknowledgeEvent *AcknowledgeEvent `json:"acknowledge_event,omitempty"`
	CheckinEvent     *CheckinEvent     `json:"checkin_event,omitempty"`
}

type AcknowledgeEvent struct {
	UDID         string            `json:"udid,omitempty"`
	EnrollmentID string            `json:"enrollment_id,omitempty"`
	Status       string            `json:"status"`
	CommandUUID  string            `json:"command_uuid,omitempty"`
	Params       map[string]string `json:"url_params,omitempty"`
	RawPayload   []byte            `json:"raw_payload"`
}

type CheckinEvent struct {
	UDID         string            `json:"udid,omitempty"`
	EnrollmentID string            `json:"enrollment_id,omitempty"`
	Params       map[string]string `json:"url_params"`
	RawPayload   []byte            `json:"raw_payload"`

	// signals which tokenupdate this is to be able to tell whether this
	// is the initial enrollment vs. a following tokenupdate
	TokenUpdateTally *int `json:"token_update_tally,omitempty"`
}
