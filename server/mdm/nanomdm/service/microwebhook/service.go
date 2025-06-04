// Package microwebhook provides a MicroMDM-emulating webhook
package microwebhook

import (
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
)

type MicroWebhook struct {
	url    string
	client *http.Client
	store  storage.TokenUpdateTallyStore
}

func New(url string, store storage.TokenUpdateTallyStore) *MicroWebhook {
	return &MicroWebhook{
		url:    url,
		client: http.DefaultClient,
		store:  store,
	}
}

func (w *MicroWebhook) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	ev := &Event{
		Topic:     "mdm.Authenticate",
		CreatedAt: time.Now(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return postWebhookEvent(r.Context, w.client, w.url, ev)
}

func (w *MicroWebhook) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	ev := &Event{
		Topic:     "mdm.TokenUpdate",
		CreatedAt: time.Now(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	if w.store != nil {
		tally, err := w.store.RetrieveTokenUpdateTally(r.Context, r.ID)
		if err != nil {
			return err
		}
		ev.CheckinEvent.TokenUpdateTally = &tally
	}
	return postWebhookEvent(r.Context, w.client, w.url, ev)
}

func (w *MicroWebhook) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	ev := &Event{
		Topic:     "mdm.CheckOut",
		CreatedAt: time.Now(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return postWebhookEvent(r.Context, w.client, w.url, ev)
}

func (w *MicroWebhook) UserAuthenticate(r *mdm.Request, m *mdm.UserAuthenticate) ([]byte, error) {
	ev := &Event{
		Topic:     "mdm.UserAuthenticate",
		CreatedAt: time.Now(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return nil, postWebhookEvent(r.Context, w.client, w.url, ev)
}

func (w *MicroWebhook) SetBootstrapToken(r *mdm.Request, m *mdm.SetBootstrapToken) error {
	ev := &Event{
		Topic:     "mdm.SetBootstrapToken",
		CreatedAt: time.Now(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return postWebhookEvent(r.Context, w.client, w.url, ev)
}

func (w *MicroWebhook) GetBootstrapToken(r *mdm.Request, m *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	ev := &Event{
		Topic:     "mdm.GetBootstrapToken",
		CreatedAt: time.Now(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return nil, postWebhookEvent(r.Context, w.client, w.url, ev)
}

func (w *MicroWebhook) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	ev := &Event{
		Topic:     "mdm.Connect",
		CreatedAt: time.Now(),
		AcknowledgeEvent: &AcknowledgeEvent{
			UDID:         results.UDID,
			EnrollmentID: results.EnrollmentID,
			Status:       results.Status,
			CommandUUID:  results.CommandUUID,
			RawPayload:   results.Raw,
			Params:       r.Params,
		},
	}
	return nil, postWebhookEvent(r.Context, w.client, w.url, ev)
}

func (w *MicroWebhook) DeclarativeManagement(r *mdm.Request, m *mdm.DeclarativeManagement) ([]byte, error) {
	ev := &Event{
		Topic:     "mdm.DeclarativeManagement",
		CreatedAt: time.Now(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return nil, postWebhookEvent(r.Context, w.client, w.url, ev)
}

func (w *MicroWebhook) GetToken(r *mdm.Request, m *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	ev := &Event{
		Topic:     "mdm.GetToken",
		CreatedAt: time.Now(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return nil, postWebhookEvent(r.Context, w.client, w.url, ev)
}
