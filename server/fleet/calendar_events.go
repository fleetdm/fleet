package fleet

import "time"

type CalendarEvent struct {
	ID        uint      `db:"id"`
	Email     string    `db:"email"`
	StartTime time.Time `db:"start_time"`
	EndTime   time.Time `db:"end_time"`
	Data      []byte    `db:"event"`

	UpdateCreateTimestamps
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
}
