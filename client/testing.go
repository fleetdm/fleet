package client

import (
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// HTTPDoer is the exported interface for an HTTP client that can perform requests.
// It mirrors the unexported httpClient interface.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewClientForTest creates a Client with the given HTTP client and base URL,
// for use in tests only. This provides a way to inject a mock HTTP client
// without access to unexported fields.
func NewClientForTest(baseURL string, httpDoer HTTPDoer) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseClient: &baseClient{
			baseURL:            u,
			http:               httpDoer,
			serverCapabilities: fleet.CapabilityMap{},
			clientCapabilities: fleet.CapabilityMap{},
		},
	}, nil
}
