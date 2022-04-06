package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"

	jira "github.com/andygrunwald/go-jira"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const (
	// JiraName is the name of the job as registered in the worker.
	JiraName = "jira"

	nvdCVEURL = "https://nvd.nist.gov/vuln/detail/"
)

var jiraSummaryTmpl = template.Must(template.New("").Parse(
	`{{ .CVE }} detected on hosts`,
))

var jiraDescriptionTmpl = template.Must(template.New("").Parse(
	`See vulnerability (CVE) details in National Vulnerability Database (NVD) here: {{ .NVDURL }}

See all hosts affected by this vulnerability (CVE) in Fleet: {{ .FleetURL }}

--

This issue was created automatically by your Fleet to Jira integration.
`,
))

type jiraTemplateArgs struct {
	CVE      string
	NVDURL   string
	FleetURL string
}

// JiraClient defines the method required for the client that makes API calls
// to Jira.
type JiraClient interface {
	CreateIssue(ctx context.Context, issue *jira.Issue) (*jira.Issue, error)
}

// Jira is the job processor for jira integrations.
type Jira struct {
	FleetURL   string
	Datastore  fleet.Datastore // TODO: we may not need the datastore, though it depends on the URL issue
	Log        kitlog.Logger
	JiraClient JiraClient
}

// Name returns the name of the job.
func (j *Jira) Name() string {
	return JiraName
}

// JiraArgs are the arguments for the Jira integration job.
type JiraArgs struct {
	CVE  string   `json:"cve"`
	CPEs []string `json:"cpes"`
}

// Run processes a worker message for the Jira integration.
func (j *Jira) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args JiraArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	// TODO(mna): as discussed as standup and explained here:
	// https://github.com/fleetdm/fleet/issues/4521#issuecomment-1090718077
	// we will create one ticket per _CPE_ for this release instead of one
	// per CVE, so that we can include the relevant link to the hosts page
	// with the corresponding software_id (the software_id associated with
	// the CPE). I think we should do this here, turning this single
	// Jira job into multiple Jira tickets. In the future, when we add the
	// CVE filter in the hosts page, we can just remove the looping here
	// and create a single ticket, no code elsewhere will have to change.

	tmplArgs := jiraTemplateArgs{
		CVE:      args.CVE,
		NVDURL:   nvdCVEURL + args.CVE,
		FleetURL: fmt.Sprintf("%s/hosts/manage?order_key=hostname&order_direction=asc&software_id=%d", j.FleetURL, 1),
	}

	var buf bytes.Buffer
	if err := jiraSummaryTmpl.Execute(&buf, &tmplArgs); err != nil {
		return ctxerr.Wrap(ctx, err, "execute summary template")
	}
	summary := buf.String()

	buf.Reset() // reuse buffer
	if err := jiraDescriptionTmpl.Execute(&buf, &tmplArgs); err != nil {
		return ctxerr.Wrap(ctx, err, "execute summary template")
	}
	description := buf.String()

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
	)

	return nil
}

// QueueJiraJobs queues the Jira vulnerability jobs to process asynchronously
// via the worker.
func QueueJiraJobs(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, recentVulns map[string][]string) error {
	level.Debug(logger).Log("enabled", "true", "recentVulns", len(recentVulns))

	for cve, cpes := range recentVulns {
		job, err := QueueJob(ctx, ds, JiraName, JiraArgs{CVE: cve, CPEs: cpes})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "queueing job")
		}
		level.Debug(logger).Log("job_id", job.ID)
	}
	return nil
}
