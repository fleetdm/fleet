package fleet

type CalendarWebhookRequest struct {
	EventUUID           string
	GoogleChannelID     string
	GoogleResourceState string
}

type CalendarWebhookResponse struct {
	Err error `json:"error,omitempty"`
}

func (r CalendarWebhookResponse) Error() error { return r.Err }
