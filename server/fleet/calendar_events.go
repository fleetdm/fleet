package fleet

import "time"

type CalendarEvent struct {
	ID        uint      `db:"id"`
	UUID      string    `db:"uuid"`
	Email     string    `db:"email"`
	StartTime time.Time `db:"start_time"`
	EndTime   time.Time `db:"end_time"`
	Data      []byte    `db:"event"`
	TimeZone  *string   `db:"timezone"` // Only nil when event existed before addition of timezone column

	UpdateCreateTimestamps
}

type CalendarEventDetails struct {
	CalendarEvent
	TeamID *uint `db:"team_id"` // Should not be nil, but is nullable in the database
	HostID uint  `db:"host_id"`
}

type CalendarWebhookStatus int

const (
	CalendarWebhookStatusNone CalendarWebhookStatus = iota
	CalendarWebhookStatusPending
	CalendarWebhookStatusSent
	CalendarWebhookStatusError
	CalendarWebhookStatusRetry
)

type HostCalendarEvent struct {
	ID              uint                  `db:"id"`
	HostID          uint                  `db:"host_id"`
	CalendarEventID uint                  `db:"calendar_event_id"`
	WebhookStatus   CalendarWebhookStatus `db:"webhook_status"`

	UpdateCreateTimestamps
}

type HostPolicyMembershipData struct {
	Email   string `db:"email"`
	Passing bool   `db:"passing"`

	HostID             uint   `db:"host_id"`
	HostDisplayName    string `db:"host_display_name"`
	HostHardwareSerial string `db:"host_hardware_serial"`
	FailingPolicyIDs   string `db:"failing_policy_ids"`
}
