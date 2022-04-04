package externalsvc

import (
	"net/http"

	"github.com/andygrunwald/go-jira"
)

// Jira is a Jira client to be used to make requests to a jira external
// service.
type Jira struct {
	client *jira.Client
}

// JiraOptions defines the options to configure a Jira client.
type JiraOptions struct {
	BaseURL           string
	BasicAuthUsername string
	BasicAuthPassword string
	Transport         http.RoundTripper
}

// NewJiraClient returns a Jira client to use to make requests to a jira
// external service.
func NewJiraClient(opts *JiraOptions) (*Jira, error) {
	basicAuth := &jira.BasicAuthTransport{
		Username:  opts.BasicAuthUsername,
		Password:  opts.BasicAuthPassword,
		Transport: opts.Transport,
	}
	client, err := jira.NewClient(basicAuth.Client(), opts.BaseURL)
	if err != nil {
		return nil, err
	}
	return &Jira{client: client}, nil
}
