package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
)

// Event is a MicroMDM webhook-ish JSON structure.
// See https://github.com/micromdm/micromdm/blob/main/docs/user-guide/api-and-webhooks.md
type Event struct {
	Topic     string    `json:"topic"`
	EventID   string    `json:"event_id"`
	CreatedAt time.Time `json:"created_at"`

	DeviceResponseEvent *DeviceResponseEvent `json:"device_response_event,omitempty"`
}

// DeviceResponseEvent represents an event for a DEP sync or fetch response.
type DeviceResponseEvent struct {
	DEPName        string                `json:"dep_name"`
	DeviceResponse *godep.DeviceResponse `json:"device_response,omitempty"`
}

// Webhook is a service that calls back to a URL with a JSON presentation of
// the DEP sync or fetch response.
type Webhook struct {
	url    string
	client *http.Client
}

// NewWebhook creates a new Webhook.
func NewWebhook(url string) *Webhook {
	return &Webhook{url: url, client: http.DefaultClient}
}

// CallWebhook assembles the JSON body from name, isFetch, and resp and calls
// to the configured service URL.
func (w *Webhook) CallWebhook(ctx context.Context, name string, isFetch bool, resp *godep.DeviceResponse) error {
	topic := "dep.SyncDevices"
	if isFetch {
		topic = "dep.FetchDevices"
	}
	event := &Event{
		Topic:     topic,
		CreatedAt: time.Now(),
		DeviceResponseEvent: &DeviceResponseEvent{
			DEPName:        name,
			DeviceResponse: resp,
		},
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	httpResp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status: %s", httpResp.Status)
	}
	return nil
}
