package worker

import (
	"context"
	"fmt"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	zendesk "github.com/nukosuke/go-zendesk/zendesk"
)

// TestAutomationFailer is an implementation of the JiraClient and ZendeskClient interfaces
// that wraps another client and introduces forced failures so that error-handling
// logic can be tested at scale in a real environment (e.g. in the load-testing
// environment).
type TestAutomationFailer struct {
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

	// ZendeskClient is the wrapped Zendesk client to use for normal calls, when no
	// forced failure is inserted.
	ZendeskClient ZendeskClient

	callCounts int
}

// CreateJiraIssue implements the JiraClient and introduces a forced failure if
// required, otherwise it returns the result of calling
// f.JiraClient.CreateJiraIssue with the provided arguments.
func (f *TestAutomationFailer) CreateJiraIssue(ctx context.Context, issue *jira.Issue) (*jira.Issue, error) {
	var testValue string
	if issue.Fields != nil && issue.Fields.Summary != "" {
		testValue = issue.Fields.Summary
	}
	if err := f.forceErr(testValue); err != nil {
		return nil, err
	}
	return f.JiraClient.CreateJiraIssue(ctx, issue)
}

// CreateZendeskTicket implements the ZendeskClient and introduces a forced failure if
// required, otherwise it returns the result of calling
// f.ZendeskClient.CreateZendeskTicket with the provided arguments.
func (f *TestAutomationFailer) CreateZendeskTicket(ctx context.Context, ticket *zendesk.Ticket) (*zendesk.Ticket, error) {
	if err := f.forceErr(ticket.Subject); err != nil {
		return nil, err
	}
	return f.ZendeskClient.CreateZendeskTicket(ctx, ticket)
}

func (f *TestAutomationFailer) JiraConfigMatches(opts *externalsvc.JiraOptions) bool {
	return f.JiraClient.JiraConfigMatches(opts)
}

func (f *TestAutomationFailer) ZendeskConfigMatches(opts *externalsvc.ZendeskOptions) bool {
	return f.ZendeskClient.ZendeskConfigMatches(opts)
}

func (f *TestAutomationFailer) forceErr(testValue string) error {
	f.callCounts++
	for _, cve := range f.AlwaysFailCVEs {
		if strings.Contains(testValue, cve) {
			return fmt.Errorf("always failing CVE %q", cve)
		}
	}
	if f.FailCallCountModulo > 0 && f.callCounts%f.FailCallCountModulo == 0 {
		return fmt.Errorf("failing due to FailCallCountModulo: callCount=%d", f.callCounts)
	}
	return nil
}
