package externalsvc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/nukosuke/go-zendesk/zendesk"
)

// Zendesk is a Zendesk client to be used to make requests to the Zendesk external service.
type Zendesk struct {
	client *zendesk.Client
	opts   ZendeskOptions
}

// ZendeskOptions defines the options to configure a Zendesk client.
type ZendeskOptions struct {
	URL      string
	Email    string
	APIToken string
	GroupID  int64
}

// NewZendeskClient returns a Zendesk client to use to make requests to the Zendesk external service.
func NewZendeskClient(opts *ZendeskOptions) (*Zendesk, error) {
	if os.Getenv("TEST_ZENDESK_CLIENT") == "true" {
		return NewZendeskTestClient(opts)
	}
	client, err := zendesk.NewClient(fleethttp.NewClient())
	if err != nil {
		return nil, err
	}

	url, err := url.Parse(opts.URL)
	if err != nil {
		return nil, err
	}
	subparts := strings.Split(url.Host, ".")
	subdomain := subparts[0]

	if err := client.SetSubdomain(subdomain); err != nil {
		return nil, err
	}
	client.SetCredential(zendesk.NewAPITokenCredential(opts.Email, opts.APIToken))

	return &Zendesk{
		client: client,
		opts:   *opts,
	}, nil
}

// GetGroup returns the group details for the group key provided in the
// Zendesk client options. It can be used to test in one request the
// authentication and connection parameters to the Zendesk instance as well as the
// existence of the group.
func (z *Zendesk) GetGroup(ctx context.Context) (*zendesk.Group, error) {
	var group *zendesk.Group

	op := func() (interface{}, error) {
		g, err := z.client.GetGroup(ctx, z.opts.GroupID)
		group = &g
		return group, err
	}

	if err := doZendeskWithRetry(op); err != nil {
		return nil, err
	}
	return group, nil
}

// CreateZendeskTicket creates a ticket on the Zendesk server targeted by the Zendesk client.
// It returns the created ticket or an error.
func (z *Zendesk) CreateZendeskTicket(ctx context.Context, ticket *zendesk.Ticket) (*zendesk.Ticket, error) {
	ticket.GroupID = z.opts.GroupID

	var createdTicket *zendesk.Ticket
	op := func() (interface{}, error) {
		t, err := z.client.CreateTicket(ctx, *ticket)
		createdTicket = &t
		return &createdTicket, err
	}

	if err := doZendeskWithRetry(op); err != nil {
		return nil, err
	}
	return createdTicket, nil
}

// ZendeskConfigMatches returns true if the zendesk client has been configured
// using those same options. The Zendesk in the name is required so that the
// interface method is not the same as the one for Jira (for mock or wrapper
// implementations).
func (z *Zendesk) ZendeskConfigMatches(opts *ZendeskOptions) bool {
	return z.opts == *opts
}

// TODO: find approach to consolidate overlapping logic for jira and zendesk retries
func doZendeskWithRetry(fn func() (interface{}, error)) error {
	op := func() error {
		_, err := fn()
		if err == nil {
			return nil
		}
		var netErr net.Error
		if errors.As(err, &netErr) {
			if netErr.Temporary() || netErr.Timeout() {
				// retryable error
				return err
			}
		}

		var zErr zendesk.Error
		if errors.As(err, &zErr) {
			statusCode := zErr.Status()
			if statusCode >= http.StatusInternalServerError {
				// 500+ status, can be worth retrying
				return err
			}
			if statusCode == http.StatusTooManyRequests {
				// handle 429 rate-limits, see
				// https://developer.zendesk.com/api-reference/ticketing/account-configuration/usage_limits/
				// for details.
				rawAfter := zErr.Headers().Get("Retry-After")
				afterSecs, err := strconv.ParseInt(rawAfter, 10, 0)
				if err == nil && (time.Duration(afterSecs)*time.Second) < maxWaitForRetryAfter {
					// the retry-after duration is reasonable, wait for it and return a
					// retryable error so that we try again.
					time.Sleep(time.Duration(afterSecs) * time.Second)
					return errors.New("retry after requested delay")
				}
			}
		}

		// at this point, this is a non-retryable error
		return backoff.Permanent(err)
	}

	boff := backoff.WithMaxRetries(backoff.NewConstantBackOff(retryBackoff), uint64(maxRetries))
	return backoff.Retry(op, boff)
}

// overrides endpoint url with full server url instead of just setting the subdomain
func NewZendeskTestClient(opts *ZendeskOptions) (*Zendesk, error) {
	client, err := zendesk.NewClient(fleethttp.NewClient())
	if err != nil {
		return nil, err
	}
	testURL := fmt.Sprint(opts.URL, "/api/v2")
	if err := client.SetEndpointURL(testURL); err != nil {
		return nil, err
	}
	client.SetCredential(zendesk.NewAPITokenCredential(opts.Email, opts.APIToken))

	return &Zendesk{
		client: client,
		opts:   *opts,
	}, nil
}
