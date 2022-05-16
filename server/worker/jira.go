package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"text/template"

	jira "github.com/andygrunwald/go-jira"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// jiraName is the name of the job as registered in the worker.
const jiraName = "jira"

var jiraTemplates = struct {
	VulnSummary              *template.Template
	VulnDescription          *template.Template
	FailingPolicySummary     *template.Template
	FailingPolicyDescription *template.Template
}{
	VulnSummary: template.Must(template.New("").Parse(
		`Vulnerability {{ .CVE }} detected on {{ len .Hosts }} host(s)`,
	)),

	// Jira uses wiki markup in the v2 api.
	VulnDescription: template.Must(template.New("").Parse(
		`See vulnerability (CVE) details in National Vulnerability Database (NVD) here: [{{ .CVE }}|{{ .NVDURL }}{{ .CVE }}].

Affected hosts:

{{ $end := len .Hosts }}{{ if gt $end 50 }}{{ $end = 50 }}{{ end }}
{{ range slice .Hosts 0 $end }}
* [{{ .Hostname }}|{{ $.FleetURL }}/hosts/{{ .ID }}]
{{ end }}

View the affected software and more affected hosts:

# Go to the [Software|{{ .FleetURL }}/software/manage] page in Fleet.
# Above the list of software, in the *Search software* box, enter "{{ .CVE }}".
# Hover over the affected software and select *View all hosts*.

----

This issue was created automatically by your Fleet Jira integration.
`)),

	FailingPolicySummary: template.Must(template.New("").Parse(
		`{{ .PolicyName }} policy failed on {{ len .Hosts }} host(s)`,
	)),

	FailingPolicyDescription: template.Must(template.New("").Parse(
		`Hosts:
{{ $end := len .Hosts }}{{ if gt $end 50 }}{{ $end = 50 }}{{ end }}
{{ range slice .Hosts 0 $end }}
* [{{ .Hostname }}|{{ $.FleetURL }}/hosts/{{ .ID }}]
{{ end }}

View hosts that failed {{ .PolicyName }} on the [*Hosts*|{{ .FleetURL }}/hosts/manage] page in Fleet.

----

This issue was created automatically by your Fleet Jira integration.
`)),
}

type jiraVulnTemplateArgs struct {
	NVDURL   string
	FleetURL string
	CVE      string
	Hosts    []*fleet.HostShort
}

// JiraClient defines the method required for the client that makes API calls
// to Jira.
type JiraClient interface {
	CreateJiraIssue(ctx context.Context, issue *jira.Issue) (*jira.Issue, error)
}

// Jira is the job processor for jira integrations.
type Jira struct {
	FleetURL   string
	Datastore  fleet.Datastore
	Log        kitlog.Logger
	JiraClient JiraClient
}

// Name returns the name of the job.
func (j *Jira) Name() string {
	return jiraName
}

// jiraArgs are the arguments for the Jira integration job.
type jiraArgs struct {
	CVE           string             `json:"cve,omitempty"`
	FailingPolicy *failingPolicyArgs `json:"failing_policy,omitempty"`
}

// Run executes the jira job.
func (j *Jira) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args jiraArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch {
	case args.CVE != "":
		return j.runVuln(ctx, args)
	//case args.FailingPolicy != nil:
	default:
		return ctxerr.New(ctx, "empty JiraArgs, nothing to process")
	}
}

func (j *Jira) runVuln(ctx context.Context, args jiraArgs) error {
	hosts, err := j.Datastore.HostsByCVE(ctx, args.CVE)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "find hosts by cve")
	}

	tmplArgs := jiraVulnTemplateArgs{
		NVDURL:   nvdCVEURL,
		FleetURL: j.FleetURL,
		CVE:      args.CVE,
		Hosts:    hosts,
	}

	var buf bytes.Buffer
	if err := jiraTemplates.VulnSummary.Execute(&buf, &tmplArgs); err != nil {
		return ctxerr.Wrap(ctx, err, "execute summary template")
	}
	summary := buf.String()

	buf.Reset() // reuse buffer
	if err := jiraTemplates.VulnDescription.Execute(&buf, &tmplArgs); err != nil {
		return ctxerr.Wrap(ctx, err, "execute description template")
	}
	description := buf.String()

	// Note, newlines get automatically escaped in json.

	issue := &jira.Issue{
		Fields: &jira.IssueFields{
			Type: jira.IssueType{
				Name: "Task",
			},
			Summary:     summary,
			Description: description,
		},
	}

	createdIssue, err := j.JiraClient.CreateJiraIssue(ctx, issue)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create issue")
	}

	level.Debug(j.Log).Log(
		"msg", "created jira issue for cve",
		"cve", args.CVE,
		"issue_id", createdIssue.ID,
		"issue_key", createdIssue.Key,
	)

	return nil
}

// QueueJiraVulnJobs queues the Jira vulnerability jobs to process asynchronously
// via the worker.
func QueueJiraVulnJobs(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, recentVulns map[string][]string) error {
	level.Info(logger).Log("enabled", "true", "recentVulns", len(recentVulns))

	// for troubleshooting, log in debug level the CVEs that we will process
	// (cannot be done in the loop below as we want to add the debug log
	// _before_ we start processing them).
	cves := make([]string, 0, len(recentVulns))
	for cve := range recentVulns {
		cves = append(cves, cve)
	}
	sort.Strings(cves)
	level.Debug(logger).Log("recent_cves", fmt.Sprintf("%v", cves))

	for cve := range recentVulns {
		job, err := QueueJob(ctx, ds, jiraName, jiraArgs{CVE: cve})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "queueing job")
		}
		level.Debug(logger).Log("job_id", job.ID)
	}
	return nil
}
