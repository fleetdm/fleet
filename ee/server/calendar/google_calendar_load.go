package calendar

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	kitlog "github.com/go-kit/log"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"io"
	"net/http"
	"net/url"
	"os"
)

// GoogleCalendarLoadAPI is used for load testing.
type GoogleCalendarLoadAPI struct {
	Logger            kitlog.Logger
	baseUrl           string
	userToImpersonate string
	ctx               context.Context
	client            *http.Client
	serverURL         string
}

// Configure creates a new Google Calendar service using the provided credentials.
func (lowLevelAPI *GoogleCalendarLoadAPI) Configure(ctx context.Context, _ string, privateKey string, userToImpersonate string,
	serverURL string) error {
	if lowLevelAPI.Logger == nil {
		lowLevelAPI.Logger = kitlog.With(kitlog.NewLogfmtLogger(os.Stderr), "mock", "GoogleCalendarLoadAPI", "user", userToImpersonate)
	}
	lowLevelAPI.baseUrl = privateKey
	lowLevelAPI.userToImpersonate = userToImpersonate
	lowLevelAPI.ctx = ctx
	if lowLevelAPI.client == nil {
		lowLevelAPI.client = fleethttp.NewClient()
	}
	lowLevelAPI.serverURL = serverURL
	return nil
}

func (lowLevelAPI *GoogleCalendarLoadAPI) GetSetting(name string) (*calendar.Setting, error) {
	reqUrl, err := url.Parse(lowLevelAPI.baseUrl + "/settings")
	if err != nil {
		return nil, err
	}
	query := reqUrl.Query()
	query.Set("name", name)
	query.Set("email", lowLevelAPI.userToImpersonate)
	reqUrl.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(lowLevelAPI.ctx, "GET", reqUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	rsp, err := lowLevelAPI.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rsp.Body.Close()
	}()
	if rsp.StatusCode != http.StatusOK {
		var data []byte
		if rsp.Body != nil {
			data, _ = io.ReadAll(rsp.Body)
		}
		return nil, fmt.Errorf("unexpected status code: %d with body: %s", rsp.StatusCode, string(data))
	}
	var setting calendar.Setting
	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &setting)
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (lowLevelAPI *GoogleCalendarLoadAPI) CreateEvent(event *calendar.Event) (*calendar.Event, error) {
	body, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	reqUrl, err := url.Parse(lowLevelAPI.baseUrl + "/events/add")
	if err != nil {
		return nil, err
	}
	query := reqUrl.Query()
	query.Set("email", lowLevelAPI.userToImpersonate)
	reqUrl.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(lowLevelAPI.ctx, "POST", reqUrl.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	rsp, err := lowLevelAPI.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rsp.Body.Close()
	}()
	if rsp.StatusCode != http.StatusCreated {
		var data []byte
		if rsp.Body != nil {
			data, _ = io.ReadAll(rsp.Body)
		}
		return nil, fmt.Errorf("unexpected status code: %d with body: %s", rsp.StatusCode, string(data))
	}
	var rspEvent calendar.Event
	body, err = io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &rspEvent)
	if err != nil {
		return nil, err
	}
	return &rspEvent, nil
}

func (lowLevelAPI *GoogleCalendarLoadAPI) UpdateEvent(event *calendar.Event) (*calendar.Event, error) {
	return nil, errors.New("GoogleCalendarLoadAPI.UpdateEvent is not implemented")
}

func (lowLevelAPI *GoogleCalendarLoadAPI) GetEvent(id, _ string) (*calendar.Event, error) {
	reqUrl, err := url.Parse(lowLevelAPI.baseUrl + "/events")
	if err != nil {
		return nil, err
	}
	query := reqUrl.Query()
	query.Set("id", id)
	reqUrl.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(lowLevelAPI.ctx, "GET", reqUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	rsp, err := lowLevelAPI.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rsp.Body.Close()
	}()
	if rsp.StatusCode == http.StatusNotFound {
		return nil, &googleapi.Error{Code: http.StatusNotFound}
	}
	if rsp.StatusCode != http.StatusOK {
		var data []byte
		if rsp.Body != nil {
			data, _ = io.ReadAll(rsp.Body)
		}
		return nil, fmt.Errorf("unexpected status code: %d with body: %s", rsp.StatusCode, string(data))
	}
	var rspEvent calendar.Event
	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &rspEvent)
	if err != nil {
		return nil, err
	}
	return &rspEvent, nil
}

func (lowLevelAPI *GoogleCalendarLoadAPI) ListEvents(timeMin string, timeMax string) (*calendar.Events, error) {
	reqUrl, err := url.Parse(lowLevelAPI.baseUrl + "/events/list")
	if err != nil {
		return nil, err
	}
	query := reqUrl.Query()
	query.Set("timemin", timeMin)
	query.Set("timemax", timeMax)
	query.Set("email", lowLevelAPI.userToImpersonate)
	reqUrl.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(lowLevelAPI.ctx, "GET", reqUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	rsp, err := lowLevelAPI.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rsp.Body.Close()
	}()
	if rsp.StatusCode != http.StatusOK {
		var data []byte
		if rsp.Body != nil {
			data, _ = io.ReadAll(rsp.Body)
		}
		return nil, fmt.Errorf("unexpected status code: %d with body: %s", rsp.StatusCode, string(data))
	}
	var events calendar.Events
	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &events)
	if err != nil {
		return nil, err
	}
	return &events, nil
}

func (lowLevelAPI *GoogleCalendarLoadAPI) DeleteEvent(id string) error {
	reqUrl, err := url.Parse(lowLevelAPI.baseUrl + "/events/delete")
	if err != nil {
		return err
	}
	query := reqUrl.Query()
	query.Set("id", id)
	reqUrl.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(lowLevelAPI.ctx, "DELETE", reqUrl.String(), nil)
	if err != nil {
		return err
	}
	rsp, err := lowLevelAPI.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = rsp.Body.Close()
	}()
	if rsp.StatusCode == http.StatusGone {
		return &googleapi.Error{Code: http.StatusGone}
	}
	if rsp.StatusCode != http.StatusOK {
		var data []byte
		if rsp.Body != nil {
			data, _ = io.ReadAll(rsp.Body)
		}
		return fmt.Errorf("unexpected status code: %d with body: %s", rsp.StatusCode, string(data))
	}
	return nil
}

func (lowLevelAPI *GoogleCalendarLoadAPI) Watch(eventUUID string, channelID string, ttl uint64) (resourceID string, err error) {
	return "resourceID", nil
}

func (lowLevelAPI *GoogleCalendarLoadAPI) Stop(channelID string, resourceID string) error {
	return nil
}
