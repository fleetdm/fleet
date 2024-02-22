package nanomdm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
)

const enrollmentIDHeader = "X-Enrollment-ID"

type DeclarativeManagementHTTPCaller struct {
	url    *url.URL
	client *http.Client
}

// NewDeclarativeManagementHTTPCaller creates a new DeclarativeManagementHTTPCaller
func NewDeclarativeManagementHTTPCaller(urlPrefix string, client *http.Client) (*DeclarativeManagementHTTPCaller, error) {
	url, err := url.Parse(urlPrefix)
	return &DeclarativeManagementHTTPCaller{url: url, client: client}, err
}

// DeclarativeManagement calls out to an HTTP URL to handle the actual Declarative Management protocol
func (c *DeclarativeManagementHTTPCaller) DeclarativeManagement(r *mdm.Request, message *mdm.DeclarativeManagement) ([]byte, error) {
	if c.url == nil {
		return nil, errors.New("missing URL")
	}
	endpointURL, err := url.Parse(message.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing endpoint URL: %w", err)
	}
	u := c.url.ResolveReference(endpointURL)
	method := http.MethodGet
	if len(message.Data) > 0 {
		method = http.MethodPut
	}
	req, err := http.NewRequestWithContext(r.Context, method, u.String(), bytes.NewBuffer(message.Data))
	if err != nil {
		return nil, err
	}
	req.Header.Set(enrollmentIDHeader, r.ID)
	if len(message.Data) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return bodyBytes, service.NewHTTPStatusError(
			resp.StatusCode,
			fmt.Errorf("unexpected HTTP status: %s", resp.Status),
		)
	}
	return bodyBytes, nil
}
