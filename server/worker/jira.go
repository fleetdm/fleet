package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"sort"

	jira "github.com/andygrunwald/go-jira"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const (
	// jiraName is the name of the job as registered in the worker.
	jiraName = "jira"
	// nvdCVEURL is the base link to a CVE on the NVD website, only the CVE code
	// needs to be appended to make it a valid link.
	nvdCVEURL = "https://nvd.nist.gov/vuln/detail/"
)

var jiraSummaryTmpl = template.Must(template.New("").Parse(
	`Vulnerability {{ .CVE }} detected on {{ len .Hosts }} host(s)`,
))

// Jira uses wiki markup in the v2 api.
var jiraDescriptionTmpl = template.Must(template.New("").Parse(
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
`,
))

type jiraTemplateArgs struct {
	NVDURL   string
	FleetURL string
	CVE      string
	Hosts    []*fleet.HostShort
}

// JiraClient defines the method required for the client that makes API calls
// to Jira.
type JiraClient interface {
	CreateIssue(ctx context.Context, issue *jira.Issue) (*jira.Issue, error)
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

// JiraArgs are the arguments for the Jira integration job.
type JiraArgs struct {
	CVE string `json:"cve"`
}

// Run executes the jira job.
func (j *Jira) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args JiraArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	hosts, err := j.Datastore.HostsByCVE(ctx, args.CVE)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "find hosts by cve")
	}

	tmplArgs := jiraTemplateArgs{
		NVDURL:   nvdCVEURL,
		FleetURL: j.FleetURL,
		CVE:      args.CVE,
		Hosts:    hosts,
	}

	var buf bytes.Buffer
	if err := jiraSummaryTmpl.Execute(&buf, &tmplArgs); err != nil {
		return ctxerr.Wrap(ctx, err, "execute summary template")
	}
	summary := buf.String()

	buf.Reset() // reuse buffer
	if err := jiraDescriptionTmpl.Execute(&buf, &tmplArgs); err != nil {
		return ctxerr.Wrap(ctx, err, "execute description template")
	}
	description := buf.String()

	// Note, newlines get automatically escaped in json.

	issue := &jira.Issue{
		Fields: &jira.IssueFields{
			Type: jira.IssueType{
				Name: "Bug",
			},
			Summary:     summary,
			Description: description,
		},
	}

	createdIssue, err := j.JiraClient.CreateIssue(ctx, issue)
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

// QueueJiraJobs queues the Jira vulnerability jobs to process asynchronously
// via the worker.
func QueueJiraJobs(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, recentVulns map[string][]string) error {
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
		job, err := QueueJob(ctx, ds, jiraName, JiraArgs{CVE: cve})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "queueing job")
		}
		level.Debug(logger).Log("job_id", job.ID)
	}
	return nil
}
