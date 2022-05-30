package externalsvc

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// Jira is a Jira client to be used to make requests to a jira external
// service.
type Jira struct {
	client *jira.Client
	opts   JiraOptions
}

// JiraOptions defines the options to configure a Jira client.
type JiraOptions struct {
	BaseURL           string
	BasicAuthUsername string
	BasicAuthPassword string
	ProjectKey        string
}

// NewJiraClient returns a Jira client to use to make requests to a jira
// external service.
func NewJiraClient(opts *JiraOptions) (*Jira, error) {
	tr := fleethttp.NewTransport()
	basicAuth := &jira.BasicAuthTransport{
		Username:  opts.BasicAuthUsername,
		Password:  opts.BasicAuthPassword,
		Transport: tr,
	}
	client, err := jira.NewClient(basicAuth.Client(), opts.BaseURL)
	if err != nil {
		return nil, err
	}

	return &Jira{
		client: client,
		opts:   *opts,
	}, nil
}

// GetProject returns the project details for the project key provided in the
// Jira client options. It can be used to test in one request the
// authentication and connection parameters to the Jira instance as well as the
// existence of the project.
func (j *Jira) GetProject(ctx context.Context) (*jira.Project, error) {
	var proj *jira.Project

	op := func() (*jira.Response, error) {
		var (
			err  error
			resp *jira.Response
		)
		proj, resp, err = j.client.Project.GetWithContext(ctx, j.opts.ProjectKey)
		return resp, err
	}

	if err := doWithRetry(op); err != nil {
		return nil, err
	}
	return proj, nil
}

// CreateJiraIssue creates an issue on the jira server targeted by the Jira client.
// It returns the created issue or an error.
func (j *Jira) CreateJiraIssue(ctx context.Context, issue *jira.Issue) (*jira.Issue, error) {
	if issue.Fields == nil {
		issue.Fields = &jira.IssueFields{}
	}
	issue.Fields.Project.Key = j.opts.ProjectKey

	var createdIssue *jira.Issue
	op := func() (*jira.Response, error) {
		var (
			err  error
			resp *jira.Response
		)
		createdIssue, resp, err = j.client.Issue.CreateWithContext(ctx, issue)
		return resp, err
	}

	if err := doWithRetry(op); err != nil {
		return nil, err
	}
	return createdIssue, nil
}

// JiraConfigMatches returns true if the jira client has been configured using
// those same options. The Jira in the name is required so that the interface
// method is not the same as the one for Zendesk (for mock or wrapper
// implementations).
func (j *Jira) JiraConfigMatches(opts *JiraOptions) bool {
	return j.opts == *opts
}

// TODO: find approach to consolidate overlapping logic for jira and zendesk retries
func doWithRetry(fn func() (*jira.Response, error)) error {
	op := func() error {
		resp, err := fn()
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

		if resp.StatusCode >= http.StatusInternalServerError {
			// 500+ status, can be worth retrying
			return err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			// handle 429 rate-limits, see
			// https://developer.atlassian.com/cloud/jira/platform/rate-limiting/
			// for details.
			rawAfter := resp.Header.Get("Retry-After")
			afterSecs, err := strconv.ParseInt(rawAfter, 10, 0)
			if err == nil && (time.Duration(afterSecs)*time.Second) < maxWaitForRetryAfter {
				// the retry-after duration is reasonable, wait for it and return a
				// retryable error so that we try again.
				time.Sleep(time.Duration(afterSecs) * time.Second)
				return errors.New("retry after requested delay")
			}
		}

		// at this point, this is a non-retryable error
		return backoff.Permanent(err)
	}

	boff := backoff.WithMaxRetries(backoff.NewConstantBackOff(retryBackoff), uint64(maxRetries))
	return backoff.Retry(op, boff)
}
