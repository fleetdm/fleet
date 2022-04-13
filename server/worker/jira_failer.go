package worker

import (
	"context"
	"fmt"
	"strings"

	jira "github.com/andygrunwald/go-jira"
)

// TestJiraFailer is an implementation of the JiraClient interface that wraps
// another JiraClient and introduces forced failures so that error-handling
// logic can be tested at scale in a real environment (e.g. in the load-testing
// environment).
type TestJiraFailer struct {
	// FailCallCountModulo is the number of calls to execute normally vs
	// forcing a failure. In other words, it will force a failure every time
	// callCounts % FailCallCountModulo == 0. If it is <= 0, no forced failure is
	// introduced based on call counts.
	FailCallCountModulo int

	// AlwaysFailCVEs is the list of CVEs for which a failure will always be
	// forced, so that those CVEs never succeed in creating a Jira ticket.
	AlwaysFailCVEs []string

	// JiraClient is the wrapped Jira client to use for normal calls, when no
	// forced failure is inserted.
	JiraClient JiraClient

	callCounts int
}

// CreateIssue implements the JiraClient and introduces a forced failure if
// required, otherwise it returns the result of calling
// f.JiraClient.CreateIssue with the provided arguments.
func (f *TestJiraFailer) CreateIssue(ctx context.Context, issue *jira.Issue) (*jira.Issue, error) {
	f.callCounts++

	if issue.Fields != nil && issue.Fields.Summary != "" {
		s := issue.Fields.Summary
		for _, cve := range f.AlwaysFailCVEs {
			if strings.Contains(s, cve) {
				return nil, fmt.Errorf("always failing CVE %q", cve)
			}
		}
	}

	if f.FailCallCountModulo > 0 && f.callCounts%f.FailCallCountModulo == 0 {
		return nil, fmt.Errorf("failing due to FailCallCountModulo: callCount=%d", f.callCounts)
	}

	return f.JiraClient.CreateIssue(ctx, issue)
}
