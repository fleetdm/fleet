// Package conditional_access_google is the Cloud Identity client for Google conditional access. It acts
// as the customer's Workspace admin through the customer's service account (domain-wide delegation).
package conditional_access_google

import (
	"context"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudidentity/v1"
	"google.golang.org/api/option"
)

// Scope is the only scope the customer delegates to their service account.
const Scope = "https://www.googleapis.com/auth/cloud-identity.devices"

// Client wraps the Cloud Identity API for one customer.
type Client struct {
	svc        *cloudidentity.Service
	customerID string
}

// NewClient builds a client that authenticates as adminEmail with a token minted from
// serviceAccountJSON, cached until it expires. endpoint overrides the Cloud Identity API
// base URL when non-empty (development and tests only). The token endpoint is the
// key's token_uri, so a mock can stand in for both.
func NewClient(ctx context.Context, serviceAccountJSON []byte, adminEmail, customerID, endpoint string) (*Client, error) {
	if customerID == "" {
		return nil, errors.New("missing customer ID")
	}
	conf, err := google.JWTConfigFromJSON(serviceAccountJSON, Scope)
	if err != nil {
		return nil, fmt.Errorf("parse service account key: %w", err)
	}
	conf.Subject = adminEmail

	// The client may outlive the request that built it, so don't tie token refreshes to its cancellation.
	tokenCtx := context.WithValue(context.WithoutCancel(ctx), oauth2.HTTPClient, fleethttp.NewClient())
	opts := []option.ClientOption{option.WithHTTPClient(conf.Client(tokenCtx))}
	if endpoint != "" {
		opts = append(opts, option.WithEndpoint(endpoint))
	}
	svc, err := cloudidentity.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create cloud identity service: %w", err)
	}
	return &Client{svc: svc, customerID: customerID}, nil
}

// CheckDelegation lists one device under the customer. An error means the key isn't delegated
// for the scope, the admin lacks the device management privilege, or the customer ID is wrong.
func (c *Client) CheckDelegation(ctx context.Context) error {
	if _, err := c.svc.Devices.List().Customer(c.customer()).PageSize(1).Context(ctx).Do(); err != nil {
		return fmt.Errorf("list devices: %w", err)
	}
	return nil
}

func (c *Client) customer() string {
	return "customers/" + c.customerID
}
