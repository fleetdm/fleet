package fleet

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

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

func (ce *CalendarEvent) GetBodyTag() string {
	type details struct {
		BodyTag string `json:"body_tag"`
	}
	var d details
	err := json.Unmarshal(ce.Data, &d)
	if err != nil {
		return ""
	}
	return d.BodyTag
}

func (ce *CalendarEvent) SaveDataItems(keysAndValues ...string) error {
	if len(keysAndValues)%2 != 0 {
		return errors.New("SaveDataItem requires an even number of arguments")
	}
	var result map[string]any
	if len(ce.Data) > 0 {
		err := json.Unmarshal(ce.Data, &result)
		if err != nil {
			return fmt.Errorf("could not unmarshal event data: %w", err)
		}
	} else {
		result = make(map[string]any, 1)
	}
	for i := 0; i < len(keysAndValues); i += 2 {
		key := keysAndValues[i]
		value := keysAndValues[i+1]
		result[key] = value
	}
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("could not marshal event data: %w", err)
	}
	ce.Data = data
	return nil
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
