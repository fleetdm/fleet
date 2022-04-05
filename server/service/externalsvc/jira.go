package externalsvc

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

const (
	maxRetries   = 5
	retryBackoff = 300 * time.Millisecond
)

// Jira is a Jira client to be used to make requests to a jira external
// service.
type Jira struct {
	client     *jira.Client
	projectKey string
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
		client:     client,
		projectKey: opts.ProjectKey,
	}, nil
}

// CurrentUser returns information about the user configured to make Jira API
// requests. It can be used to test authentication and connection parameters
// to the Jira instance.
func (j *Jira) CurrentUser(ctx context.Context) (*jira.User, error) {
	var user *jira.User

	op := func() (*jira.Response, error) {
		var (
			err  error
			resp *jira.Response
		)
		user, resp, err = j.client.User.GetSelfWithContext(ctx)
		return resp, err
	}

	if err := doWithRetry(op); err != nil {
		return nil, err
	}
	return user, nil
}

// CreateIssue creates an issue on the jira server targeted by the Jira client.
// It returns the created issue or an error.
func (j *Jira) CreateIssue(ctx context.Context, issue *jira.Issue) (*jira.Issue, error) {
	if issue.Fields == nil {
		issue.Fields = &jira.IssueFields{}
	}
	issue.Fields.Project.Key = j.projectKey

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

		// at this point, this is a non-retryable error
		return backoff.Permanent(err)
	}

	boff := backoff.WithMaxRetries(backoff.NewConstantBackOff(retryBackoff), uint64(maxRetries))
	return backoff.Retry(op, boff)
}
